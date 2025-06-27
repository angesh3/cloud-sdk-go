/**
 * Custom error handler for pxGrid Cloud Demo
 * This file specifically handles console error suppression for known issues
 */
(function() {
    // Store original console methods
    const originalConsoleError = console.error;
    
    // Override console.error to filter out specific error messages
    console.error = function() {
        // Convert arguments to a string for pattern matching
        const errorText = Array.from(arguments).join(' ');
        
        // Filter out known error patterns
        if (errorText.includes('Error disconnecting device')) {
            // Silently suppress this specific error
            return;
        }
        
        // Pass through all other errors to the original console.error
        return originalConsoleError.apply(console, arguments);
    };
    
    console.log('Error handler initialized');
})();
