// main.js
document.addEventListener('DOMContentLoaded', function() {
    const submitButton = document.querySelector('input[type="submit"]');
    submitButton.addEventListener('click', function(event) {
        // Simple validation example
        const queryText = document.getElementById('query_text').value;
        if (queryText.trim() === '') {
            alert('Please enter a query.');
            event.preventDefault();
        }
    });
});
