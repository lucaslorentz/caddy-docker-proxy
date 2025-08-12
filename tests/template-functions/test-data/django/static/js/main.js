// Django Static JavaScript - Test File
console.log('Django static JS loaded');

document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded - Django app ready');
    
    // Simulate Django static file functionality
    const staticIndicator = document.createElement('div');
    staticIndicator.textContent = 'Static JS loaded from: ' + window.location.origin;
    staticIndicator.style.cssText = 'position: fixed; top: 10px; right: 10px; background: #417690; color: white; padding: 5px; font-size: 12px; border-radius: 3px;';
    document.body.appendChild(staticIndicator);
});