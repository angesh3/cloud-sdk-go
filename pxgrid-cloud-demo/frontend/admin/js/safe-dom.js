/**
 * Safe DOM Manipulation Helper
 * Provides utilities for safely manipulating DOM elements with proper checks
 */

// Safely get DOM element, returns null if not found
function safeGetElement(id) {
    return document.getElementById(id);
}

// Safely set text content of an element
function safeSetText(id, text) {
    const element = safeGetElement(id);
    if (element) {
        element.textContent = text;
        return true;
    }
    console.warn(`Element with ID '${id}' not found, cannot set text content`);
    return false;
}

// Safely add event listener to an element
function safeAddEventListener(id, event, handler) {
    const element = safeGetElement(id);
    if (element) {
        element.addEventListener(event, handler);
        return true;
    }
    console.warn(`Element with ID '${id}' not found, cannot add event listener`);
    return false;
}

// Safely show an element
function safeShowElement(id) {
    const element = safeGetElement(id);
    if (element) {
        element.style.display = '';
        return true;
    }
    return false;
}

// Safely hide an element
function safeHideElement(id) {
    const element = safeGetElement(id);
    if (element) {
        element.style.display = 'none';
        return true;
    }
    return false;
}

// Safely update UI with user info
function updateUserUI(user) {
    if (!user) return;
    
    // Update username display in header
    safeSetText('username-display', user.username);
    
    // Any other user info updates
}
