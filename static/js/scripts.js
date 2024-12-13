document.addEventListener('DOMContentLoaded', function() {
    var body = document.body;
    var editMode = false;
    var originalMetadata = {};
    var currentFilePath = ''; // Initialize as an empty string

    // Function to set theme
    function setTheme(theme) {
        requestAnimationFrame(() => {
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
            applyThemeToDrawer(theme); // Apply theme to drawer
        });
    }

    // Function to apply theme to drawer
    function applyThemeToDrawer(theme) {
        var drawer = document.getElementById('fileMetadataDrawer');
        if (theme === 'dark') {
            drawer.classList.add('dark-theme');
            drawer.classList.remove('light-theme');
        } else {
            drawer.classList.add('light-theme');
            drawer.classList.remove('dark-theme');
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

    // Initialize upload modal logic
    var uploadFilesInput = document.getElementById('uploadFiles');
    var sameVersionCheckbox = document.getElementById('sameVersionCheckbox');
    var singleVersionField = document.getElementById('singleVersionField');
    var perFileVersionFields = document.getElementById('perFileVersionFields');

    // Handle the "Use the same version for all files" checkbox
    if (sameVersionCheckbox) {
        sameVersionCheckbox.addEventListener('change', function() {
            if (sameVersionCheckbox.checked) {
                singleVersionField.style.display = 'block';
                perFileVersionFields.style.display = 'none';
            } else {
                singleVersionField.style.display = 'none';
                perFileVersionFields.style.display = 'block';
                generatePerFileVersionFields();
            }
        });
    }

    // Generate per-file version input fields
    function generatePerFileVersionFields() {
        var files = uploadFilesInput.files;
        perFileVersionFields.innerHTML = '';
        for (var i = 0; i < files.length; i++) {
            var file = files[i];

            var div = document.createElement('div');
            div.classList.add('input-field');

            // Hidden input for filename
            var hiddenInput = document.createElement('input');
            hiddenInput.type = 'hidden';
            hiddenInput.name = 'fileNames';
            hiddenInput.value = file.name;

            // Input for version
            var input = document.createElement('input');
            input.type = 'text';
            input.name = 'fileVersions';
            input.required = true;
            input.id = 'fileVersion_' + i;

            var label = document.createElement('label');
            label.htmlFor = 'fileVersion_' + i;
            label.textContent = 'Version for ' + file.name;

            div.appendChild(hiddenInput);
            div.appendChild(input);
            div.appendChild(label);

            perFileVersionFields.appendChild(div);
        }

        if (typeof M !== 'undefined') {
            M.updateTextFields(); // Update labels
        }
    }

    // Handle file selection change
    if (uploadFilesInput) {
        uploadFilesInput.addEventListener('change', function() {
            if (!sameVersionCheckbox.checked) {
                generatePerFileVersionFields();
            }
        });
    }

    // Function to handle unauthorized responses
    function handleUnauthorizedResponse(response) {
        if (response.status === 401) {
            requestAnimationFrame(() => {
                var loginModal = M.Modal.getInstance(document.getElementById('loginModal'));
                loginModal.open();
            });
            return true;
        }
        return false;
    }

    // Function to check login status before performing an action
    function checkLoginAndPerformAction(action) {
        fetch('/check-session', {
            method: 'GET',
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                action();
            } else {
                var loginModal = M.Modal.getInstance(document.getElementById('loginModal'));
                loginModal.open();
            }
        }).catch(error => {
            console.error('Error checking session:', error);
            var loginModal = M.Modal.getInstance(document.getElementById('loginModal'));
            loginModal.open();
        });
    }

    // Handle upload form submission
    var uploadForm = document.getElementById('uploadForm');
    if (uploadForm) {
        uploadForm.addEventListener('submit', function (event) {
            event.preventDefault();
            checkLoginAndPerformAction(function() {
                var sameVersionCheckbox = document.getElementById('sameVersionCheckbox');
                var perFileVersionFields = document.getElementById('perFileVersionFields');

                // If "same version" checkbox is checked, remove `required` attributes from per-file fields
                if (sameVersionCheckbox && sameVersionCheckbox.checked) {
                    var fileVersionInputs = perFileVersionFields.querySelectorAll('input[name="fileVersions"]');
                    fileVersionInputs.forEach(function (input) {
                        input.removeAttribute('required');
                    });
                }

                // Add file paths to the form data
                var formData = new FormData(uploadForm);
                var files = uploadFilesInput.files;
                for (var i = 0; i < files.length; i++) {
                    formData.append('filePaths', files[i].webkitRelativePath || files[i].name);
                }

                // Submit the form with file paths
                fetch(uploadForm.action, {
                    method: 'POST',
                    body: formData,
                }).then(response => {
                    if (handleUnauthorizedResponse(response)) return;
                    if (response.ok) {
                        window.location.reload();
                    } else {
                        response.text().then(text => {
                            M.toast({ html: 'Error uploading files: ' + text });
                        });
                    }
                }).catch(error => {
                    console.error('Error uploading files:', error);
                    M.toast({ html: 'Error uploading files' });
                });
            });
        });
    }

    // Handle delete form submission
    var deleteForm = document.getElementById('deleteForm');
    if (deleteForm) {
        deleteForm.addEventListener('submit', function (event) {
            event.preventDefault();
            checkLoginAndPerformAction(function() {
                var formData = new FormData(deleteForm);
                fetch(deleteForm.action, {
                    method: 'POST',
                    body: formData,
                }).then(response => {
                    if (handleUnauthorizedResponse(response)) return;
                    if (response.ok) {
                        window.location.reload();
                    } else {
                        response.text().then(text => {
                            M.toast({ html: 'Error deleting files: ' + text });
                        });
                    }
                }).catch(error => {
                    console.error('Error deleting files:', error);
                    M.toast({ html: 'Error deleting files' });
                });
            });
        });
    }

    // Handle rename form submission
    var renameForm = document.getElementById('renameForm');
    if (renameForm) {
        renameForm.addEventListener('submit', function (event) {
            event.preventDefault();
            checkLoginAndPerformAction(function() {
                var formData = new FormData(renameForm);
                fetch(renameForm.action, {
                    method: 'POST',
                    body: formData,
                }).then(response => {
                    if (handleUnauthorizedResponse(response)) return;
                    if (response.ok) {
                        window.location.reload();
                    } else {
                        response.text().then(text => {
                            M.toast({ html: 'Error renaming file: ' + text });
                        });
                    }
                }).catch(error => {
                    console.error('Error renaming file:', error);
                    M.toast({ html: 'Error renaming file' });
                });
            });
        });
    }

    // Handle move form submission
    var moveForm = document.getElementById('moveForm');
    if (moveForm) {
        moveForm.addEventListener('submit', function (event) {
            event.preventDefault();
            checkLoginAndPerformAction(function() {
                var formData = new FormData(moveForm);
                fetch(moveForm.action, {
                    method: 'POST',
                    body: formData,
                }).then(response => {
                    if (handleUnauthorizedResponse(response)) return;
                    if (response.ok) {
                        window.location.reload();
                    } else {
                        response.text().then(text => {
                            M.toast({ html: 'Error moving files: ' + text });
                        });
                    }
                }).catch(error => {
                    console.error('Error moving files:', error);
                    M.toast({ html: 'Error moving files' });
                });
            });
        });
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

            // Manage download button
            if (downloadButton) {
                if (anyChecked) {
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

                checkLoginAndPerformAction(function() {
                    // Populate the delete confirmation modal with the names of the items to be deleted
                    var deleteItemsList = document.getElementById('deleteItemsList');
                    deleteItemsList.innerHTML = '';
                    var checkedItems = document.querySelectorAll('.item-checkbox:checked');
                    checkedItems.forEach(function(checkbox) {
                        var li = document.createElement('li');
                        li.textContent = checkbox.value;
                        deleteItemsList.appendChild(li);
                    });

                    // Set the current path in the hidden input field
                    var deleteCurrentPath = document.getElementById('deleteCurrentPath');
                    deleteCurrentPath.value = document.querySelector('input[name="currentPath"]').value;

                    var modal = M.Modal.getInstance(document.getElementById('deleteConfirmModal'));
                    modal.open();
                });
            });
        }

        // Confirm delete button handler
        document.getElementById('confirmDeleteButton').addEventListener('click', function() {
            var deleteCurrentPath = document.getElementById('deleteCurrentPath').value;
            fileForm.action = '/delete';
            fileForm.method = 'post';

            // Create a hidden input to include the current path in the form submission
            var currentPathInput = document.createElement('input');
            currentPathInput.type = 'hidden';
            currentPathInput.name = 'currentPath';
            currentPathInput.value = deleteCurrentPath;
            fileForm.appendChild(currentPathInput);

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
                checkLoginAndPerformAction(function() {
                    var modal = M.Modal.getInstance(document.getElementById('uploadModal'));
                    modal.open();
                });
            });
        }

        // Add authorization check before showing create folder modal
        var createFolderButton = document.getElementById('createFolderButton');
        if (createFolderButton) {
            createFolderButton.addEventListener('click', function(event) {
                event.preventDefault();
                checkLoginAndPerformAction(function() {
                    var modal = M.Modal.getInstance(document.getElementById('createFolderModal'));
                    modal.open();
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
                checkLoginAndPerformAction(function() {
                    var checkedItems = document.querySelectorAll('.item-checkbox:checked');
                    if (checkedItems.length === 1) {
                        var itemPath = checkedItems[0].value;
                        var itemName = itemPath.split('/').pop();
                        document.getElementById('renameOldPath').value = itemPath;
                        document.getElementById('newName').value = itemName;
                        document.getElementById('newNameLabel').classList.add('active');
                        var modal = M.Modal.getInstance(document.getElementById('renameModal'));
                        modal.open();
                    } else {
                        M.toast({html: 'Please select exactly one item to rename.'});
                    }
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
                checkLoginAndPerformAction(function() {
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

    // Function to open drawer
    function openDrawer() {
        requestAnimationFrame(() => {
            var drawer = document.getElementById('fileMetadataDrawer');
            var backdrop = document.createElement('div');
            backdrop.className = 'drawer-backdrop open';
            backdrop.style.zIndex = '999'; // Ensure backdrop is behind the drawer
            backdrop.addEventListener('click', function() {
                if (editMode) {
                    var modal = M.Modal.getInstance(document.getElementById('confirmCloseDrawerModal'));
                    modal.open();
                } else {
                    closeDrawer();
                }
            });
            document.body.appendChild(backdrop);
            drawer.classList.add('open');
            drawer.style.zIndex = '1000'; // Ensure drawer is in front of the backdrop

            // Reset edit mode switch
            var editModeSwitch = document.getElementById('editModeSwitch');
            if (editModeSwitch.checked) {
                editModeSwitch.checked = false;
                toggleEditMode();
            }
            // Correctly populate RDS Number
            document.getElementById('rdsNumber').value = originalMetadata['RDS RDS'] || '';
        });
    }

    // Function to close drawer
    function closeDrawer() {
        var drawer = document.getElementById('fileMetadataDrawer');
        var backdrop = document.querySelector('.drawer-backdrop');
        if (backdrop) {
            backdrop.remove();
        }
        drawer.classList.remove('open');
        if (editMode) {
            toggleEditMode();
        }
        editMode = false; // Reset edit mode
        var editModeSwitch = document.getElementById('editModeSwitch');
        if (editModeSwitch) {
            editModeSwitch.checked = false; // Uncheck the switch
        }
    }

    // Function to copy text to clipboard
    function copyToClipboard(text) {
        navigator.clipboard.writeText(text).then(function() {
            M.toast({html: 'Copied to clipboard'});
        }).catch(function(error) {
            console.error('Error copying to clipboard:', error);
        });
    }

    // Function to toggle edit mode
    function toggleEditMode() {
        editMode = !editMode;
        var metadataContent = document.getElementById('fileMetadataContent');
        var metadataEditForm = document.getElementById('metadataEditForm');
        if (editMode) {
            metadataContent.style.display = 'none';
            metadataEditForm.style.display = 'block';
        } else {
            metadataContent.style.display = 'block';
            metadataEditForm.style.display = 'none';
        }
    }

    // Function to check if user is logged in
    function checkLoginStatus(callback) {
        fetch('/check-session', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => {
            if (response.ok) {
                callback(true);
            } else {
                callback(false);
            }
        })
        .catch(error => {
            console.error('Error checking session:', error);
            callback(false);
        });
    }

    // Event listener for edit mode switch
    var editModeSwitch = document.getElementById('editModeSwitch');
    if (editModeSwitch) {
        editModeSwitch.addEventListener('change', function() {
            if (editMode) {
                var modal = M.Modal.getInstance(document.getElementById('confirmCloseDrawerModal'));
                modal.open();
            } else {
                checkLoginStatus(function(isLoggedIn) {
                    if (isLoggedIn) {
                        toggleEditMode();
                    } else {
                        var loginModal = M.Modal.getInstance(document.getElementById('loginModal'));
                        loginModal.open();
                    }
                });
            }
        });
    }

    // Event listener for save button
    var saveMetadataButton = document.getElementById('saveMetadataButton');
    if (saveMetadataButton) {
        saveMetadataButton.addEventListener('click', function () {
            const rdsNumber = document.getElementById('rdsNumber').value;
            const rdsCRC32 = document.getElementById('rdsCRC32').value;
            const rdsSHA1 = document.getElementById('rdsSHA1').value;
            const rdsSHA256 = document.getElementById('rdsSHA256').value;
            const rdsCRC64 = document.getElementById('rdsCRC64').value.toUpperCase();
            const rdsBLAKE2sp = document.getElementById('rdsBLAKE2sp').value;
            const version = document.getElementById('version').value;

            // Use the currentFilePath variable
            const filePath = currentFilePath;

            // Сравнение с оригинальными данными
            const updatedMetadata = {};
            if (originalMetadata['RDS RDS'] !== rdsNumber) updatedMetadata['RDS RDS'] = rdsNumber;
            if (originalMetadata['RDS CRC32'] !== rdsCRC32) updatedMetadata['RDS CRC32'] = rdsCRC32;
            if (originalMetadata['RDS CRC64'] !== rdsCRC64) updatedMetadata['RDS CRC64'] = rdsCRC64;
            if (originalMetadata['RDS BLAKE2sp'] !== rdsBLAKE2sp) updatedMetadata['RDS BLAKE2sp'] = rdsBLAKE2sp;
            if (originalMetadata['RDS SHA1'] !== rdsSHA1) updatedMetadata['RDS SHA1'] = rdsSHA1;
            if (originalMetadata['RDS SHA256'] !== rdsSHA256) updatedMetadata['RDS SHA256'] = rdsSHA256;
            if (originalMetadata['Version'] !== version) updatedMetadata['Version'] = version;

            if (Object.keys(updatedMetadata).length === 0) {
                M.toast({ html: 'No changes to save' });
                return;
            }

            updatedMetadata['FilePath'] = filePath;

            fetch('/save-metadata', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(updatedMetadata),
            })
                .then(response => {
                    if (response.ok) {
                        M.toast({ html: 'Metadata saved successfully' });
                        if (editMode) {
                            toggleEditMode(); // Reset edit mode
                        }
                        closeDrawer(); // Close the drawer after saving
                        return; // Add return statement to prevent further execution
                    } else {
                        M.toast({ html: 'Error saving metadata' });
                    }
                })
                .catch(error => {
                    console.error('Error saving metadata:', error);
                    M.toast({ html: 'Error saving metadata' });
                });
        });
    }

    // Event listener for cancel button
    var cancelMetadataButton = document.getElementById('cancelMetadataButton');
    if (cancelMetadataButton) {
        cancelMetadataButton.addEventListener('click', function() {
            if (editMode) {
                var modal = M.Modal.getInstance(document.getElementById('confirmCloseDrawerModal'));
                modal.open();
            } else {
                toggleEditMode();
            }
        });
    }

    // Function to copy hashes to clipboard
    function copyHashes() {
        var hashes = document.querySelectorAll('.hash-field');
        var hashText = '';
        hashes.forEach(function(hash) {
            hashText += hash.textContent + '\n';
        });
        navigator.clipboard.writeText(hashText).then(function() {
            M.toast({html: 'Hashes copied to clipboard'});
        }).catch(function(error) {
            console.error('Error copying hashes:', error);
        });
    }

    // Event listener for copy hashes button
    var copyHashesButton = document.getElementById('copyHashesButton');
    if (copyHashesButton) {
        copyHashesButton.addEventListener('click', copyHashes);
    }

    // Event listener for file info icons
    var fileInfoIcons = document.querySelectorAll('.file-info-icon');
    fileInfoIcons.forEach(function(icon) {
        icon.addEventListener('click', function(event) {
            event.preventDefault();
            var filePath = this.getAttribute('data-file');
            currentFilePath = filePath; // Set the current file path
            
            fetch('/file-metadata?path=' + encodeURIComponent(filePath))
                .then(response => response.json())
                .then(data => {
                    originalMetadata = data;
                    var metadataContent = document.getElementById('fileMetadataContent');
                    metadataContent.innerHTML = ''; // Clear previous content

                    // Create General group
                    var generalGroup = document.createElement('div');
                    generalGroup.classList.add('metadata-group');
                    var generalHeader = document.createElement('h5');
                    generalHeader.textContent = 'General';
                    generalGroup.appendChild(generalHeader);

                    // Uploader
                    var uploaderDiv = document.createElement('div');
                    uploaderDiv.classList.add('metadata-field');
                    var uploaderLabel = document.createElement('label');
                    uploaderLabel.textContent = 'Uploader:';
                    var uploaderValue = document.createElement('input');
                    uploaderValue.type = 'text';
                    uploaderValue.value = data['Uploader'];
                    uploaderValue.readOnly = true;
                    uploaderValue.classList.add('metadata-input');
                    uploaderDiv.appendChild(uploaderLabel);
                    uploaderDiv.appendChild(uploaderValue);
                    generalGroup.appendChild(uploaderDiv);

                    // Version
                    var versionDiv = document.createElement('div');
                    versionDiv.classList.add('metadata-field');
                    var versionLabel = document.createElement('label');
                    versionLabel.textContent = 'Version:';
                    var versionValue = document.createElement('input');
                    versionValue.type = 'text';
                    versionValue.value = data['Version'];
                    versionValue.readOnly = true;
                    versionValue.classList.add('metadata-input');
                    versionDiv.appendChild(versionLabel);
                    versionDiv.appendChild(versionValue);
                    generalGroup.appendChild(versionDiv);

                    // Append General group to metadata content
                    metadataContent.appendChild(generalGroup);

                    // Create Hashes group
                    var hashesGroup = document.createElement('div');
                    hashesGroup.classList.add('metadata-group');
                    var hashesHeader = document.createElement('h5');
                    hashesHeader.textContent = 'Hashes';
                    hashesGroup.appendChild(hashesHeader);

                    // Hash fields
                    var hashes = ['CRC32', 'SHA1', 'SHA256', 'CRC64', 'BLAKE2sp'];
                    hashes.forEach(function(hash) {
                        var hashDiv = document.createElement('div');
                        hashDiv.classList.add('metadata-field');
                        var hashLabel = document.createElement('label');
                        hashLabel.textContent = hash + ':';
                        var hashValue = document.createElement('input');
                        hashValue.type = 'text';
                        hashValue.value = data[hash];
                        hashValue.readOnly = true;
                        hashValue.classList.add('metadata-input');
                        var copyIcon = document.createElement('i');
                        copyIcon.className = 'material-icons copy-icon';
                        copyIcon.textContent = 'content_copy';
                        copyIcon.addEventListener('click', function() {
                            copyToClipboard(data[hash]);
                        });
                        hashDiv.appendChild(hashLabel);
                        var hashContainer = document.createElement('div');
                        hashContainer.classList.add('hash-container');
                        hashContainer.appendChild(hashValue);
                        hashContainer.appendChild(copyIcon);
                        hashDiv.appendChild(hashContainer);
                        hashesGroup.appendChild(hashDiv);
                    });

                    // Append Hashes group to metadata content
                    metadataContent.appendChild(hashesGroup);

                    // Create RDS group
                    var rdsGroup = document.createElement('div');
                    rdsGroup.classList.add('metadata-group');
                    var rdsHeader = document.createElement('h5');
                    rdsHeader.textContent = 'RDS';
                    rdsGroup.appendChild(rdsHeader);

                    // RDS Number (always displayed)
                    var rdsNumberDiv = document.createElement('div');
                    rdsNumberDiv.classList.add('metadata-field');
                    var rdsNumberLabel = document.createElement('label');
                    rdsNumberLabel.textContent = 'RDS Number:';
                    var rdsNumberValue = document.createElement('input');
                    rdsNumberValue.type = 'text';
                    rdsNumberValue.value = data['RDS RDS'] || ''; // Correctly populate RDS Number
                    rdsNumberValue.readOnly = true;
                    rdsNumberValue.classList.add('metadata-input');
                    rdsNumberDiv.appendChild(rdsNumberLabel);
                    rdsNumberDiv.appendChild(rdsNumberValue);
                    rdsGroup.appendChild(rdsNumberDiv);

                    // Other RDS fields with comparison status
                    var rdsFields = [
                        { field: 'RDS CRC32', compareWith: 'CRC32' },
                        { field: 'RDS CRC64', compareWith: 'CRC64' },
                        { field: 'RDS BLAKE2sp', compareWith: 'BLAKE2sp' },
                        { field: 'RDS SHA1', compareWith: 'SHA1' },
                        { field: 'RDS SHA256', compareWith: 'SHA256' }
                    ];

                    rdsFields.forEach(function(item) {
                        var rdsValue = data[item.field] || '';
                        if (rdsValue) {
                            var rdsDiv = document.createElement('div');
                            rdsDiv.classList.add('metadata-field');
                            var rdsLabel = document.createElement('label');

                            // Create status icon if compareWith is specified
                            if (item.compareWith) {
                                var statusIcon = document.createElement('i');
                                var originalValue = data[item.compareWith];

                                if (!originalValue || !rdsValue) {
                                    // Either value is empty or undefined; display grey icon
                                    statusIcon.className = 'material-icons status-icon grey-text';
                                    statusIcon.textContent = 'help_outline';
                                } else if (originalValue === rdsValue) {
                                    // Values match; display green checkmark
                                    statusIcon.className = 'material-icons status-icon green-text';
                                    statusIcon.textContent = 'check_circle';
                                } else {
                                    // Values do not match; display red cross
                                    statusIcon.className = 'material-icons status-icon red-text';
                                    statusIcon.textContent = 'cancel';
                                }
                                rdsLabel.appendChild(statusIcon);
                            }

                            rdsLabel.appendChild(document.createTextNode(' ' + item.field + ':'));

                            var rdsValueInput = document.createElement('input');
                            rdsValueInput.type = 'text';
                            rdsValueInput.value = rdsValue;
                            rdsValueInput.readOnly = true;
                            rdsValueInput.classList.add('metadata-input');
                            rdsDiv.appendChild(rdsLabel);
                            rdsDiv.appendChild(rdsValueInput);
                            rdsGroup.appendChild(rdsDiv);
                        }
                    });

                    metadataContent.appendChild(rdsGroup);

                    // Populate edit form
                    document.getElementById('rdsNumber').value = data['RDS RDS'] || '';
                    document.getElementById('rdsCRC32').value = data['RDS CRC32'] || '';
                    document.getElementById('rdsCRC64').value = data['RDS CRC64'] ? data['RDS CRC64'].toUpperCase() : '';
                    document.getElementById('rdsBLAKE2sp').value = data['RDS BLAKE2sp'] || '';
                    document.getElementById('rdsSHA1').value = data['RDS SHA1'] || '';
                    document.getElementById('rdsSHA256').value = data['RDS SHA256'] || '';
                    document.getElementById('version').value = data['Version'] || '';

                    openDrawer();
                })
                .catch(error => {
                    console.error('Ошибка при получении метаданных файла:', error);
                });
        });
    });

    // Event listener for confirm close drawer button
    var confirmCloseDrawerButton = document.getElementById('confirmCloseDrawerButton');
    if (confirmCloseDrawerButton) {
        confirmCloseDrawerButton.addEventListener('click', function() {
            var modal = M.Modal.getInstance(document.getElementById('confirmCloseDrawerModal'));
            modal.close();
            closeDrawer();
        });
    }

    // Function to get the logged-in user's name
    function getLoggedInUsername() {
        return fetch('/check-session', {
            method: 'GET',
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => data.username)
        .catch(error => {
            console.error('Error fetching username:', error);
            return '';
        });
    }

    // Initialize clipboard.js for copy functionality
    if (typeof ClipboardJS !== 'undefined') {
        new ClipboardJS('.metadata-input');
    }

    // Initialize clipboard.js for copy functionality
    new ClipboardJS('.metadata-input');

    // Function to refresh metadata
    function refreshMetadata(filePath) {
        if (window.Worker) {
            const worker = new Worker('/static/js/metadataWorker.js');
            worker.postMessage({ filePath });

            worker.onmessage = function(event) {
                const { hashes, metadata } = event.data;
                // Update metadata fields with new hashes
                document.getElementById('rdsCRC32').value = hashes['CRC32'];
                document.getElementById('rdsSHA1').value = hashes['SHA1'];
                document.getElementById('rdsSHA256').value = hashes['SHA256'];
                document.getElementById('rdsCRC64').value = hashes['CRC64'].toUpperCase();
                document.getElementById('rdsBLAKE2sp').value = hashes['BLAKE2sp'];

                // Fetch updated metadata from README.md
                fetch('/file-metadata?path=' + encodeURIComponent(filePath))
                    .then(response => response.json())
                    .then(metadata => {
                        document.getElementById('rdsNumber').value = metadata['RDS RDS'] || '';
                        M.toast({ html: 'Metadata refreshed successfully' });

                        // Update the displayed metadata content
                        updateMetadataContent(metadata, hashes);
                    })
                    .catch(error => {
                        console.error('Error fetching metadata:', error);
                        M.toast({ html: 'Error refreshing metadata' });
                    });
            };

            worker.onerror = function(error) {
                console.error('Error in metadata worker:', error);
                M.toast({ html: 'Error refreshing metadata' });
            };
        } else {
            // Fallback if Web Workers are not supported
            fetch('/recalculate-hashes?path=' + encodeURIComponent(filePath))
                .then(response => response.json())
                .then(hashes => {
                    // Update metadata fields with new hashes
                    document.getElementById('rdsCRC32').value = hashes['CRC32'];
                    document.getElementById('rdsSHA1').value = hashes['SHA1'];
                    document.getElementById('rdsSHA256').value = hashes['SHA256'];
                    document.getElementById('rdsCRC64').value = hashes['CRC64'].toUpperCase();
                    document.getElementById('rdsBLAKE2sp').value = hashes['BLAKE2sp'];

                    // Fetch updated metadata from README.md
                    fetch('/file-metadata?path=' + encodeURIComponent(filePath))
                        .then(response => response.json())
                        .then(metadata => {
                            document.getElementById('rdsNumber').value = metadata['RDS RDS'] || '';
                            M.toast({ html: 'Metadata refreshed successfully' });

                            // Update the displayed metadata content
                            updateMetadataContent(metadata, hashes);
                        })
                        .catch(error => {
                            console.error('Error fetching metadata:', error);
                            M.toast({ html: 'Error refreshing metadata' });
                        });
                })
                .catch(error => {
                    console.error('Error recalculating hashes:', error);
                    M.toast({ html: 'Error refreshing metadata' });
                });
        }
    }

    // Function to update metadata content
    function updateMetadataContent(metadata, hashes) {
        var metadataContent = document.getElementById('fileMetadataContent');
        metadataContent.innerHTML = ''; // Clear previous content

        // Create General group
        var generalGroup = document.createElement('div');
        generalGroup.classList.add('metadata-group');
        var generalHeader = document.createElement('h5');
        generalHeader.textContent = 'General';
        generalGroup.appendChild(generalHeader);

        // Uploader
        var uploaderDiv = document.createElement('div');
        uploaderDiv.classList.add('metadata-field');
        var uploaderLabel = document.createElement('label');
        uploaderLabel.textContent = 'Uploader:';
        var uploaderValue = document.createElement('input');
        uploaderValue.type = 'text';
        uploaderValue.value = metadata['Uploader'];
        uploaderValue.readOnly = true;
        uploaderValue.classList.add('metadata-input');
        uploaderDiv.appendChild(uploaderLabel);
        uploaderDiv.appendChild(uploaderValue);
        generalGroup.appendChild(uploaderDiv);

        // Version
        var versionDiv = document.createElement('div');
        versionDiv.classList.add('metadata-field');
        var versionLabel = document.createElement('label');
        versionLabel.textContent = 'Version:';
        var versionValue = document.createElement('input');
        versionValue.type = 'text';
        versionValue.value = metadata['Version'];
        versionValue.readOnly = true;
        versionValue.classList.add('metadata-input');
        versionDiv.appendChild(versionLabel);
        versionDiv.appendChild(versionValue);
        generalGroup.appendChild(versionDiv);

        // Append General group to metadata content
        metadataContent.appendChild(generalGroup);

        // Create Hashes group
        var hashesGroup = document.createElement('div');
        hashesGroup.classList.add('metadata-group');
        var hashesHeader = document.createElement('h5');
        hashesHeader.textContent = 'Hashes';
        hashesGroup.appendChild(hashesHeader);

        // Hash fields
        var hashFields = ['CRC32', 'SHA1', 'SHA256', 'CRC64', 'BLAKE2sp'];
        hashFields.forEach(function(hash) {
            var hashDiv = document.createElement('div');
            hashDiv.classList.add('metadata-field');
            var hashLabel = document.createElement('label');
            hashLabel.textContent = hash + ':';
            var hashValue = document.createElement('input');
            hashValue.type = 'text';
            hashValue.value = hashes[hash];
            hashValue.readOnly = true;
            hashValue.classList.add('metadata-input');
            var copyIcon = document.createElement('i');
            copyIcon.className = 'material-icons copy-icon';
            copyIcon.textContent = 'content_copy';
            copyIcon.addEventListener('click', function() {
                copyToClipboard(hashes[hash]);
            });
            hashDiv.appendChild(hashLabel);
            var hashContainer = document.createElement('div');
            hashContainer.classList.add('hash-container');
            hashContainer.appendChild(hashValue);
            hashContainer.appendChild(copyIcon);
            hashDiv.appendChild(hashContainer);
            hashesGroup.appendChild(hashDiv);
        });

        // Append Hashes group to metadata content
        metadataContent.appendChild(hashesGroup);

        // Create RDS group
        var rdsGroup = document.createElement('div');
        rdsGroup.classList.add('metadata-group');
        var rdsHeader = document.createElement('h5');
        rdsHeader.textContent = 'RDS';
        rdsGroup.appendChild(rdsHeader);

        // RDS Number (always displayed)
        var rdsNumberDiv = document.createElement('div');
        rdsNumberDiv.classList.add('metadata-field');
        var rdsNumberLabel = document.createElement('label');
        rdsNumberLabel.textContent = 'RDS Number:';
        var rdsNumberValue = document.createElement('input');
        rdsNumberValue.type = 'text';
        rdsNumberValue.value = metadata['RDS RDS'] || ''; // Correctly populate RDS Number
        rdsNumberValue.readOnly = true;
        rdsNumberValue.classList.add('metadata-input');
        rdsNumberDiv.appendChild(rdsNumberLabel);
        rdsNumberDiv.appendChild(rdsNumberValue);
        rdsGroup.appendChild(rdsNumberDiv);

        // Other RDS fields with comparison status
        var rdsFields = [
            { field: 'RDS CRC32', compareWith: 'CRC32' },
            { field: 'RDS CRC64', compareWith: 'CRC64' },
            { field: 'RDS BLAKE2sp', compareWith: 'BLAKE2sp' },
            { field: 'RDS SHA1', compareWith: 'SHA1' },
            { field: 'RDS SHA256', compareWith: 'SHA256' }
        ];

        rdsFields.forEach(function(item) {
            var rdsValue = metadata[item.field] || '';
            if (rdsValue) {
                var rdsDiv = document.createElement('div');
                rdsDiv.classList.add('metadata-field');
                var rdsLabel = document.createElement('label');

                // Create status icon if compareWith is specified
                if (item.compareWith) {
                    var statusIcon = document.createElement('i');
                    var originalValue = hashes[item.compareWith];

                    if (!originalValue || !rdsValue) {
                        // Either value is empty or undefined; display grey icon
                        statusIcon.className = 'material-icons status-icon grey-text';
                        statusIcon.textContent = 'help_outline';
                    } else if (originalValue === rdsValue) {
                        // Values match; display green checkmark
                        statusIcon.className = 'material-icons status-icon green-text';
                        statusIcon.textContent = 'check_circle';
                    } else {
                        // Values do not match; display red cross
                        statusIcon.className = 'material-icons status-icon red-text';
                        statusIcon.textContent = 'cancel';
                    }
                    rdsLabel.appendChild(statusIcon);
                }

                rdsLabel.appendChild(document.createTextNode(' ' + item.field + ':'));

                var rdsValueInput = document.createElement('input');
                rdsValueInput.type = 'text';
                rdsValueInput.value = rdsValue;
                rdsValueInput.readOnly = true;
                rdsValueInput.classList.add('metadata-input');
                rdsDiv.appendChild(rdsLabel);
                rdsDiv.appendChild(rdsValueInput);
                rdsGroup.appendChild(rdsDiv);
            }
        });

        metadataContent.appendChild(rdsGroup);
    }

    // Event listener for refresh metadata button
    var refreshMetadataButton = document.getElementById('refreshMetadataButton');
    if (refreshMetadataButton) {
        refreshMetadataButton.addEventListener('click', function() {
            refreshMetadata(currentFilePath);
        });
    }

    function updateRDSStatus() {
        var rows = document.querySelectorAll('#fileTable tbody tr');
        rows.forEach(function(row) {
            var fileName = row.querySelector('td a.file-link')?.getAttribute('data-file');
            if (fileName) {
                fetch('/file-metadata?path=' + encodeURIComponent(fileName))
                    .then(response => response.json())
                    .then(metadata => {
                        var rdsStatusCell = row.querySelector('.rds-status');
                        if (metadata['RDS CRC32'] === metadata['CRC32'] ||
                            metadata['RDS CRC64'] === metadata['CRC64'] ||
                            metadata['RDS SHA1'] === metadata['SHA1'] ||
                            metadata['RDS SHA256'] === metadata['SHA256'] ||
                            metadata['RDS BLAKE2sp'] === metadata['BLAKE2sp']) {
                            rdsStatusCell.innerHTML = '<i class="material-icons green-text">check_circle</i>';
                        } else if (metadata['RDS CRC32'] || metadata['RDS CRC64'] || metadata['RDS SHA1'] || metadata['RDS SHA256'] || metadata['RDS BLAKE2sp']) {
                            rdsStatusCell.innerHTML = '<i class="material-icons red-text">cancel</i>';
                        } else if (!fileName.endsWith('.md') && !fileName.endsWith('.html') && !fileName.endsWith('.txt')) {
                            rdsStatusCell.innerHTML = '<i class="material-icons grey-text">help_outline</i>';
                        } else {
                            rdsStatusCell.innerHTML = '<i class="material-icons" style="visibility: hidden;">help_outline</i>';
                        }
                    })
                    .catch(error => {
                        console.error('Error fetching metadata:', error);
                    });
            }
        });
    }

    updateRDSStatus();

    // Handle login form submission
    var loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', function(event) {
            event.preventDefault();
            var formData = new FormData(loginForm);
            fetch('/login', {
                method: 'POST',
                body: formData,
            }).then(response => {
                if (response.ok) {
                    window.location.reload();
                } else {
                    response.text().then(text => {
                        var loginError = document.getElementById('loginError');
                        loginError.textContent = text;
                        loginError.style.display = 'block';
                    });
                }
            }).catch(error => {
                console.error('Error logging in:', error);
                var loginError = document.getElementById('loginError');
                loginError.textContent = 'Error logging in';
                loginError.style.display = 'block';
            });
        });
    }

    // Scripts moved from index.html
    var editReadmeButton = document.getElementById('editReadmeButton');
    var editReadmeModal = document.getElementById('editReadmeModal');
    var previewReadmeButton = document.getElementById('previewReadmeButton');
    var editReadmeButtonFromPreview = document.getElementById('editReadmeButtonFromPreview');
    var readmePreview = document.getElementById('readmePreview');
    var saveReadmeButton = document.getElementById('saveReadmeButton');
    var cancelReadmeButton = document.getElementById('cancelReadmeButton');
    var readmeEditor = document.getElementById('readmeEditor');
    var editMode = document.getElementById('editMode');
    var previewMode = document.getElementById('previewMode');

    var readmeModalInstance = M.Modal.getInstance(editReadmeModal);

    if (editReadmeButton) {
        editReadmeButton.addEventListener('click', function() {
            checkLoginAndPerformAction(function() {
                // Reset to edit mode when opening the modal
                editMode.style.display = 'block';
                previewMode.style.display = 'none';
                readmeModalInstance.open();
            });
        });
    }

    if (previewReadmeButton) {
        previewReadmeButton.addEventListener('click', function() {
            var markdownContent = readmeEditor.value;
            fetch('/preview-markdown', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ content: markdownContent })
            })
            .then(response => response.text())
            .then(html => {
                readmePreview.innerHTML = html;
                // Switch to preview mode
                editMode.style.display = 'none';
                previewMode.style.display = 'block';
            });
        });
    }

    if (editReadmeButtonFromPreview) {
        editReadmeButtonFromPreview.addEventListener('click', function() {
            // Switch back to edit mode
            previewMode.style.display = 'none';
            editMode.style.display = 'block';
        });
    }

    if (saveReadmeButton) {
        saveReadmeButton.addEventListener('click', function() {
            var markdownContent = readmeEditor.value;
            var currentPath = document.querySelector('input[name="currentPath"]').value;
            fetch('/save-readme?path=' + encodeURIComponent(currentPath), {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ content: markdownContent })
            })
            .then(response => {
                if (response.ok) {
                    location.reload();
                } else {
                    alert('Error saving README.md');
                }
            });
        });
    }

    if (cancelReadmeButton) {
        cancelReadmeButton.addEventListener('click', function() {
            readmeModalInstance.close();
        });
    }

    var createReadmeButton = document.getElementById('createReadmeButton');
    if (createReadmeButton) {
        createReadmeButton.addEventListener('click', function() {
            checkLoginAndPerformAction(function() {
                // Reset to edit mode when opening the modal
                document.getElementById('editMode').style.display = 'block';
                document.getElementById('previewMode').style.display = 'none';
                var readmeModalInstance = M.Modal.getInstance(document.getElementById('editReadmeModal'));
                readmeModalInstance.open();
            });
        });
    }
});

document.getElementById('previewReadmeButton').addEventListener('click', function() {
    document.getElementById('editMode').style.display = 'none';
    document.getElementById('previewMode').style.display = 'block';
    document.getElementById('previewReadmeButton').style.display = 'none';
    document.getElementById('editReadmeButtonFromPreview').style.display = 'inline-block';
});

document.getElementById('editReadmeButtonFromPreview').addEventListener('click', function() {
    document.getElementById('editMode').style.display = 'block';
    document.getElementById('previewMode').style.display = 'none';
    document.getElementById('previewReadmeButton').style.display = 'inline-block';
    document.getElementById('editReadmeButtonFromPreview').style.display = 'none';
});

document.getElementById('toggleFullscreenButton').addEventListener('click', function() {
    const modal = document.getElementById('editReadmeModal');
    const modalContent = modal.querySelector('.modal-content');
    if (modal.classList.contains('fullscreen')) {
        modal.classList.remove('fullscreen');
        modalContent.classList.remove('fullscreen');
        this.innerHTML = '<i class="material-icons">fullscreen</i>';
    } else {
        modal.classList.add('fullscreen');
        modalContent.classList.add('fullscreen');
        this.innerHTML = '<i class="material-icons">fullscreen_exit</i>';
    }
});