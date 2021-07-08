package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/caddyserver/caddy/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/fsnotify/fsnotify"
	"github.com/lucaslorentz/caddy-docker-proxy/plugin/v2/caddyfile"

	"go.uber.org/zap"
)

func (g *CaddyfileGenerator) getServiceTemplatedCaddyfile(service *swarm.Service, logger *zap.Logger) (*caddyfile.Container, error) {
	err := setupTemplateDirWatcher(logger)
	if err != nil {
		logger.Info("no template dir to watch", zap.Error(err))
		// don't exit, we'll try again later..
	}

	matcher := strings.TrimPrefix(service.Spec.Name, "/")
	logger.Debug("getServiceTemplatedCaddyfile", zap.String("matcher", matcher), zap.String("labels", fmt.Sprintf("%v", service.Spec.Labels)))

	funcMap := template.FuncMap{
		"entitytype": func(options ...interface{}) (string, error) {
			return "service", nil
		},
		"upstreams": func(options ...interface{}) (string, error) {
			targets, err := g.getServiceProxyTargets(service, logger, true)
			transformed := []string{}
			for _, target := range targets {
				for _, param := range options {
					if protocol, isProtocol := param.(string); isProtocol {
						target = protocol + "://" + target
					} else if port, isPort := param.(int); isPort {
						target = target + ":" + strconv.Itoa(port)
					}
				}
				transformed = append(transformed, target)
			}
			logger.Debug("getServiceTemplatedCaddyfile", zap.Strings("upstreams", transformed))

			return strings.Join(transformed, " "), err
		},
		"matcher": func(options ...interface{}) (string, error) {
			return matcher, nil
		},
		"labels": func(options ...interface{}) (map[string]string, error) {
			return service.Spec.Labels, nil
		},
		"hostname": func(options ...interface{}) (string, error) {
			// if there is a string param, use it.
			if len(options) == 1 {
				if host, isString := options[0].(string); isString && host != "" {
					return host, nil
				}
			}
			return strings.TrimPrefix(service.Spec.Name, "/"), nil
		},
	}
	return g.getTemplatedCaddyfile(service, funcMap, logger)
}

func (g *CaddyfileGenerator) getContainerTemplatedCaddyfile(container *types.Container, logger *zap.Logger) (*caddyfile.Container, error) {
	err := setupTemplateDirWatcher(logger)
	if err != nil {
		logger.Info("no template dir to watch", zap.Error(err))
		// don't exit, we'll try again later..
	}

	name := container.ID
	if len(container.Names) > 0 {
		name = container.Names[0]
	}
	matcher := strings.TrimPrefix(name, "/")
	logger.Debug("getContainerTemplatedCaddyfile", zap.String("matcher", matcher), zap.String("labels", fmt.Sprintf("%v", container.Labels)))

	funcMap := template.FuncMap{
		"entitytype": func(options ...interface{}) (string, error) {
			return "container", nil
		},
		"upstreams": func(options ...interface{}) (string, error) {
			targets, err := g.getContainerIPAddresses(container, logger, true)
			transformed := []string{}
			for _, target := range targets {
				for _, param := range options {
					if protocol, isProtocol := param.(string); isProtocol {
						target = protocol + "://" + target
					} else if port, isPort := param.(int); isPort {
						target = target + ":" + strconv.Itoa(port)
					}
				}
				transformed = append(transformed, target)
			}
			logger.Debug("getContainerTemplatedCaddyfile", zap.Strings("upstreams", transformed))
			return strings.Join(transformed, " "), err
		},
		"matcher": func(options ...interface{}) (string, error) {
			return matcher, nil
		},
		"labels": func(options ...interface{}) (map[string]string, error) {
			return container.Labels, nil
		},
		"hostname": func(options ...interface{}) (string, error) {
			// if there is a string param, use it.
			if len(options) == 1 {
				if host, isString := options[0].(string); isString && host != "" {
					return host, nil
				}
			}
			return strings.TrimPrefix(name, "/"), nil
		},
	}
	return g.getTemplatedCaddyfile(container, funcMap, logger)
}

type tmplData struct {
	name string
	tmpl string
}

var loadedTemplates *template.Template
var newTemplate chan tmplData
var templateDirWatcher *fsnotify.Watcher

// NewTemplate adds a new named template to the parsing queue
func NewTemplate(name, tmpl string) {
	newTemplate <- tmplData{
		name: name,
		tmpl: tmpl,
	}
}

func init() {
	newTemplate = make(chan tmplData, 20)

	commonFuncMap := template.FuncMap{
		"http": func() string {
			return "http"
		},
		"https": func() string {
			return "https"
		},
	}
	loadedTemplates = template.New("").Funcs(sprig.TxtFuncMap()).Funcs(commonFuncMap)

}

func (g *CaddyfileGenerator) getTemplatedCaddyfile(data interface{}, funcMap template.FuncMap, logger *zap.Logger) (*caddyfile.Container, error) {
	loadedTemplates = loadedTemplates.Funcs(funcMap)

	// Parse any found or updated templates TMPL: prefix is to diferentiate from funcMap / named templates
	for {
		select {
		case tmpl := <-newTemplate:
			logger.Debug("parsing template", zap.String("name", tmpl.name))

			t := loadedTemplates.New("TMPL:" + tmpl.name)
			_, err := t.Parse(tmpl.tmpl)
			if err != nil {
				logger.Error("parsing template", zap.String("name", tmpl.name), zap.Error(err))
			}
		default:
			// no changed templates found
			goto noTemplates
		}
	}
noTemplates:

	var block caddyfile.Container
	for _, tmpl := range loadedTemplates.Templates() {
		if !strings.HasPrefix(tmpl.Name(), "TMPL:") {
			continue
		}
		var writer bytes.Buffer
		err := loadedTemplates.ExecuteTemplate(&writer, tmpl.Name(), data)
		if err != nil {
			logger.Error("ExecuteTemplate", zap.String("name", tmpl.Name()), zap.Error(err))
			continue
		}

		newblock, err := caddyfile.Unmarshal(writer.Bytes())
		if err != nil {
			logger.Error("problem converting template to caddyfile block", zap.String("name", tmpl.Name()), zap.Error(err))
			continue
		}
		block.Merge(newblock)
	}

	return &block, nil
}

func setupTemplateDirWatcher(logger *zap.Logger) error {
	if templateDirWatcher != nil {
		// Already initialised
		return nil
	}
	// watch for templates in "${XDG_CONFIG_HOME}/caddy/docker-proxy/"
	rootDir := filepath.Join(caddy.AppConfigDir(), "docker-proxy")
	cleanRoot := filepath.Clean(rootDir)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("docker-proxy template dir (%s) is not a Directory", cleanRoot)
	}

	templateDirWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal("Failed to create watcher", zap.Error(err))
	}
	go func() {
		for {
			select {
			case event, ok := <-templateDirWatcher.Events:
				if !ok {
					logger.Info("Stopping watching for filesystem changes")
					return
				}
				if !strings.HasSuffix(event.Name, ".tmpl") {
					logger.Debug("ignoring non .tmpl file", zap.String("name", event.Name))
					continue
				}

				removeBytes := []byte("## removed " + event.Name + " file\n\n")
				b := removeBytes
				b, err = ioutil.ReadFile(event.Name)
				if err != nil {
					logger.Error("reading event, will remove from templates", zap.String("name", event.Name), zap.Error(err))

					b = removeBytes
				}

				NewTemplate(event.Name, string(b))
			case err, ok := <-templateDirWatcher.Errors:
				if !ok {
					logger.Info("Stopping watching for filesystem changes")
					return
				}
				logger.Error("watcher error", zap.Error(err))
			}
		}
	}()

	err = templateDirWatcher.Add(cleanRoot)
	if err != nil {
		logger.Error("watcher error", zap.String("dir", cleanRoot), zap.Error(err))
		logger.Fatal("watcher error", zap.String("dir", cleanRoot), zap.Error(err))
	}
	logger.Info("Watching for updates to files ending with .tmp", zap.String("dir", cleanRoot))

	// Also need to read the existing files
	err = filepath.Walk(cleanRoot, func(path string, info os.FileInfo, e1 error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".tmpl") {
			logger.Debug("found template file", zap.String("file", path))

			if e1 != nil {
				logger.Error("problem walking dir", zap.Error(e1))
				return nil // continue with other files
			}

			b, e2 := ioutil.ReadFile(path)
			if e2 != nil {
				logger.Error("problem reading file", zap.String("file", path), zap.Error(e1))
				return nil // continue with other files
			}
			NewTemplate(path, string(b))
		}
		return nil
	})
	return nil
}
