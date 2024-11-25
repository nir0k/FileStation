document.addEventListener('DOMContentLoaded', function() {
    var body = document.body;

    // Function to set theme
    function setTheme(theme) {
        if (theme === 'dark') {
            body.classList.remove('light-theme');
            body.classList.add('dark-theme');
        } else {
            body.classList.remove('dark-theme');
            body.classList.add('light-theme');
        }
        localStorage.setItem('theme', theme);
        if (typeof M !== 'undefined') {
            M.updateTextFields(); // Reinitialize input labels
        }
    }

    // Get saved theme from localStorage
    var savedTheme = localStorage.getItem('theme') || 'light';
    setTheme(savedTheme);

    // Theme toggle functionality
    var themeToggle = document.getElementById('themeToggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', function(event) {
            event.preventDefault();
            var currentTheme = body.classList.contains('dark-theme') ? 'dark' : 'light';
            var newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            setTheme(newTheme);
        });
    }

    // Initialize modals
    var modals = document.querySelectorAll('.modal');
    if (typeof M !== 'undefined') {
        M.Modal.init(modals);
    }

    // Initialize tooltips
    var tooltippedElements = document.querySelectorAll('.tooltipped');
    if (typeof M !== 'undefined') {
        M.Tooltip.init(tooltippedElements);
    }

    // Initialize checkboxes (only if they exist on the page)
    var selectAllCheckbox = document.getElementById('selectAll');
    var itemCheckboxes = document.querySelectorAll('.item-checkbox');
    var downloadButton = document.getElementById('downloadButton');
    var deleteButton = document.getElementById('deleteButton');
    var renameButton = document.getElementById('renameButton');
    var moveButton = document.getElementById('moveButton');
    var fileForm = document.getElementById('fileForm');

    // Proceed only if file management elements are present
    if (fileForm) {
        // Function to get cookie value by name
        function getCookie(name) {
            let matches = document.cookie.match(new RegExp(
                '(?:^|; )' + name.replace(/([$?*|{}()[]\\\/+^])/g, '\\$1') + '=([^;]*)'
            ));
            return matches ? decodeURIComponent(matches[1]) : undefined;
        }

        // Display error message if present in cookie
        var errorMessage = getCookie('ErrorMessage');
        if (errorMessage) {
            M.toast({html: errorMessage, displayLength: 5000});
            // Remove the cookie after displaying the message
            document.cookie = 'ErrorMessage=; Max-Age=0; path=/';
        }

        function updateButtons() {
            var checkedItems = document.querySelectorAll('.item-checkbox:checked');
            var anyChecked = checkedItems.length > 0;
            var anyFileChecked = document.querySelectorAll('.item-checkbox[data-type="file"]:checked').length > 0;

            // Manage download button
            if (downloadButton) {
                if (anyFileChecked) {
                    downloadButton.classList.remove('disabled');
                } else {
                    downloadButton.classList.add('disabled');
                }
            }

            // Manage delete button
            if (deleteButton) {
                if (anyChecked) {
                    deleteButton.classList.remove('disabled');
                } else {
                    deleteButton.classList.add('disabled');
                }
            }

            // Manage move button
            if (moveButton) {
                if (checkedItems.length >= 1) {
                    moveButton.classList.remove('disabled');
                } else {
                    moveButton.classList.add('disabled');
                }
            }

            // Manage rename button
            if (renameButton) {
                if (checkedItems.length === 1) {
                    renameButton.classList.remove('disabled');
                } else {
                    renameButton.classList.add('disabled');
                }
            }
        }

        if (selectAllCheckbox) {
            selectAllCheckbox.addEventListener('change', function() {
                var checked = this.checked;
                itemCheckboxes.forEach(function(checkbox) {
                    checkbox.checked = checked;
                });
                updateButtons();
            });
        }

        itemCheckboxes.forEach(function(checkbox) {
            checkbox.addEventListener('change', function() {
                if (!this.checked && selectAllCheckbox) {
                    selectAllCheckbox.checked = false;
                } else if (document.querySelectorAll('.item-checkbox:checked').length === itemCheckboxes.length && selectAllCheckbox) {
                    selectAllCheckbox.checked = true;
                }
                updateButtons();
            });
        });

        // Download button handler
        if (downloadButton) {
            downloadButton.addEventListener('click', function(event) {
                if (downloadButton.classList.contains('disabled')) {
                    event.preventDefault();
                    return;
                }
                // Form submission handled by form's submit event
                fileForm.action = '/download';
                fileForm.method = 'post';
                fileForm.submit();
            });
        }

        // Delete button handler with modal confirmation
        if (deleteButton) {
            deleteButton.addEventListener('click', function(event) {
                event.preventDefault();
                if (deleteButton.classList.contains('disabled')) {
                    return;
                }

                // Populate the delete confirmation modal with the names of the items to be deleted
                var deleteItemsList = document.getElementById('deleteItemsList');
                deleteItemsList.innerHTML = '';
                var checkedItems = document.querySelectorAll('.item-checkbox:checked');
                checkedItems.forEach(function(checkbox) {
                    var li = document.createElement('li');
                    li.textContent = checkbox.value;
                    deleteItemsList.appendChild(li);
                });

                var modal = M.Modal.getInstance(document.getElementById('deleteConfirmModal'));
                modal.open();
            });
        }

        // Confirm delete button handler
        document.getElementById('confirmDeleteButton').addEventListener('click', function() {
            fileForm.action = '/delete';
            fileForm.method = 'post';
            fileForm.submit();
        });

        // Update button states on page load
        updateButtons();

        // Initialize column resizing
        var thElements = document.querySelectorAll('th.resizable');
        thElements.forEach(function(th) {
            var handle = th.querySelector('.resize-handle');
            if (handle) {
                handle.addEventListener('mousedown', initResize);
            }
        });

        function initResize(e) {
            var th = e.target.parentElement;
            var startX = e.clientX;
            var startWidth = th.offsetWidth;

            function doResize(e) {
                var newWidth = startWidth + (e.clientX - startX);
                th.style.width = newWidth + 'px';
            }

            function stopResize(e) {
                document.removeEventListener('mousemove', doResize);
                document.removeEventListener('mouseup', stopResize);
                // Save column width
                saveColumnWidths();
            }

            document.addEventListener('mousemove', doResize);
            document.addEventListener('mouseup', stopResize);
        }

        // Function to save column widths
        function saveColumnWidths() {
            var widths = [];
            thElements.forEach(function(th) {
                widths.push(th.offsetWidth);
            });
            localStorage.setItem('columnWidths', JSON.stringify(widths));
        }

        // Function to load column widths
        function loadColumnWidths() {
            var widths = JSON.parse(localStorage.getItem('columnWidths'));
            if (widths) {
                thElements.forEach(function(th, index) {
                    th.style.width = widths[index] + 'px';
                });
            }
        }

        // Load saved column widths
        loadColumnWidths();

        // Add authorization check before showing upload modal
        var uploadButton = document.getElementById('uploadFilesButton');
        if (uploadButton) {
            uploadButton.addEventListener('click', function(event) {
                event.preventDefault();
                fetch('/check-session', {
                    method: 'GET',
                    credentials: 'include'
                }).then(response => {
                    if (response.ok) {
                        // Open modal if authorized
                        var modal = M.Modal.getInstance(document.getElementById('uploadModal'));
                        modal.open();
                    } else {
                        // Redirect to login if not authorized
                        window.location.href = '/login';
                    }
                }).catch(error => {
                    console.error('Error checking session:', error);
                    window.location.href = '/login';
                });
            });
        }

        // Add authorization check before showing create folder modal
        var createFolderButton = document.getElementById('createFolderButton');
        if (createFolderButton) {
            createFolderButton.addEventListener('click', function(event) {
                event.preventDefault();
                fetch('/check-session', {
                    method: 'GET',
                    credentials: 'include'
                }).then(response => {
                    if (response.ok) {
                        // Open modal if authorized
                        var modal = M.Modal.getInstance(document.getElementById('createFolderModal'));
                        modal.open();
                    } else {
                        // Redirect to login if not authorized
                        window.location.href = '/login';
                    }
                }).catch(error => {
                    console.error('Error checking session:', error);
                    window.location.href = '/login';
                });
            });
        }

        // Add authorization check before showing rename modal
        if (renameButton) {
            renameButton.addEventListener('click', function(event) {
                event.preventDefault();
                if (renameButton.classList.contains('disabled')) {
                    return;
                }
                fetch('/check-session', {
                    method: 'GET',
                    credentials: 'include'
                }).then(response => {
                    if (response.ok) {
                        var checkedItems = document.querySelectorAll('.item-checkbox:checked');
                        if (checkedItems.length === 1) {
                            var itemPath = checkedItems[0].value;
                            document.getElementById('renameOldPath').value = itemPath;
                            var modal = M.Modal.getInstance(document.getElementById('renameModal'));
                            modal.open();
                        } else {
                            M.toast({html: 'Please select exactly one item to rename.'});
                        }
                    } else {
                        window.location.href = '/login';
                    }
                }).catch(error => {
                    console.error('Error checking session:', error);
                    window.location.href = '/login';
                });
            });
        }

        // Handler for the "Move" button
        if (moveButton) {
            moveButton.addEventListener('click', function(event) {
                event.preventDefault();
                if (moveButton.classList.contains('disabled')) {
                    return;
                }
                fetch('/check-session', {
                    method: 'GET',
                    credentials: 'include'
                }).then(response => {
                    if (response.ok) {
                        var checkedItems = document.querySelectorAll('.item-checkbox:checked');
                        if (checkedItems.length > 0) {
                            // Get the list of selected item paths
                            var itemPaths = [];
                            checkedItems.forEach(function(checkbox) {
                                itemPaths.push(checkbox.value);
                            });
                            // Save the list in a hidden form field
                            document.getElementById('moveItemPaths').value = JSON.stringify(itemPaths);

                            // Display the list of selected items
                            displaySelectedItems(itemPaths);

                            // Initialize folder navigation
                            initFolderNavigation('/');
                        } else {
                            M.toast({html: 'Please select at least one item to move.'});
                        }
                    } else {
                        window.location.href = '/login';
                    }
                }).catch(error => {
                    console.error('Error checking session:', error);
                    window.location.href = '/login';
                });
            });

            // Function to display the list of selected items
            function displaySelectedItems(itemPaths) {
                var selectedItemsList = document.getElementById('selectedItemsList');
                selectedItemsList.innerHTML = '<p>Selected Items:</p>';
                var ul = document.createElement('ul');
                itemPaths.forEach(function(path) {
                    var li = document.createElement('li');
                    li.textContent = path;
                    ul.appendChild(li);
                });
                selectedItemsList.appendChild(ul);
            }

            // Function to initialize folder navigation
            function initFolderNavigation(currentPath) {
                // Update breadcrumbs
                updateBreadcrumbs(currentPath);

                // Load folder list
                fetch('/list-folders?path=' + encodeURIComponent(currentPath))
                    .then(response => response.json())
                    .then(data => {
                        var folderList = document.getElementById('folderList');
                        folderList.innerHTML = '';

                        // Add item to go up one level if not at root
                        if (currentPath !== '/') {
                            var upItem = document.createElement('li');
                            upItem.classList.add('collection-item', 'folder-item');
                            upItem.innerHTML = '<i class="material-icons">arrow_upward</i> Go Up';
                            upItem.addEventListener('click', function() {
                                var parentPath = currentPath.substring(0, currentPath.lastIndexOf('/')) || '/';
                                initFolderNavigation(parentPath);
                            });
                            folderList.appendChild(upItem);
                        }

                        data.folders.forEach(function(folder) {
                            var li = document.createElement('li');
                            li.classList.add('collection-item', 'folder-item');
                            li.innerHTML = '<i class="material-icons">folder</i> ' + folder;

                            // Define newPath here
                            var newPath = currentPath === '/' ? '/' + folder : currentPath + '/' + folder;

                            li.addEventListener('click', function() {
                                // Navigate into folder
                                document.getElementById('selectedDestinationPath').value = newPath;
                                document.getElementById('confirmMoveButton').disabled = false;
                            });

                            li.addEventListener('dblclick', function() {
                                // Select current folder as destination
                                initFolderNavigation(newPath);
                                document.getElementById('selectedDestinationPath').value = newPath;
                                document.getElementById('confirmMoveButton').disabled = false; 
                            });
                            folderList.appendChild(li);
                        });

                        // Open modal after loading folder list
                        var modal = M.Modal.getInstance(document.getElementById('moveModal'));
                        modal.open();

                        // Disable confirm button until a folder is selected
                        document.getElementById('confirmMoveButton').disabled = true;
                    });
            }

            // Function to update breadcrumbs
            function updateBreadcrumbs(currentPath) {
                var breadcrumbs = document.getElementById('moveBreadcrumbs');
                breadcrumbs.innerHTML = '';

                var pathParts = currentPath.split('/').filter(function(part) { return part !== ''; });
                var fullPath = '';
                var isRoot = currentPath === '/';

                // Add "Home" to breadcrumbs
                var homeBreadcrumb = document.createElement('a');
                homeBreadcrumb.href = '#!';
                homeBreadcrumb.classList.add('breadcrumb');
                homeBreadcrumb.textContent = 'Home';
                homeBreadcrumb.addEventListener('click', function() {
                    initFolderNavigation('/');
                });
                breadcrumbs.appendChild(homeBreadcrumb);

                pathParts.forEach(function(part, index) {
                    fullPath += '/' + part;
                    var crumb = document.createElement('a');
                    crumb.href = '#!';
                    crumb.classList.add('breadcrumb');
                    crumb.textContent = part;
                    crumb.addEventListener('click', function() {
                        initFolderNavigation(fullPath);
                    });
                    breadcrumbs.appendChild(crumb);
                });
            }

            // Close modal on successful move
            var moveModalForm = document.querySelector('#moveModal form');
            moveModalForm.addEventListener('submit', function() {
                var modal = M.Modal.getInstance(document.getElementById('moveModal'));
                modal.close();
            });
        } // End of moveButton check
    } // End of fileForm check

    var elems = document.querySelectorAll('.modal');
    var instances = M.Modal.init(elems);
    
    document.getElementById('confirmDeleteButton').addEventListener('click', function() {
        document.getElementById('fileForm').submit();
    });
});