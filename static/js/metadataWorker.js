self.onmessage = function(event) {
    const { filePath } = event.data;

    fetch('/recalculate-hashes?path=' + encodeURIComponent(filePath))
        .then(response => response.json())
        .then(hashes => {
            self.postMessage({ hashes });
        })
        .catch(error => {
            console.error('Error recalculating hashes:', error);
            self.postMessage({ error: 'Error recalculating hashes' });
        });
};
