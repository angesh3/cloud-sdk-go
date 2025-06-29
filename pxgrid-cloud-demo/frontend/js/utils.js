/**
 * Utility functions for pxGrid Cloud Demo
 */

// Utility function to handle field details safely - must be defined globally
window.hiddenFieldDetails = function(obj, field, defaultValue = 'N/A') {
    if (!obj) return defaultValue;
    
    try {
        // Handle nested properties with dot notation (e.g., 'user.name')
        if (field && field.includes && field.includes('.')) {
            const parts = field.split('.');
            let current = obj;
            
            for (let i = 0; i < parts.length; i++) {
                if (current === null || current === undefined || typeof current !== 'object') {
                    return defaultValue;
                }
                current = current[parts[i]];
            }
            
            return current !== undefined && current !== null ? current : defaultValue;
        }
        
        // Simple property access
        return obj[field] !== undefined && obj[field] !== null ? obj[field] : defaultValue;
    } catch (e) {
        console.log('Error in hiddenFieldDetails:', e);
        return defaultValue;
    }
};

// Log that the utilities file has loaded
console.log('Utilities loaded - hiddenFieldDetails function is available globally');
