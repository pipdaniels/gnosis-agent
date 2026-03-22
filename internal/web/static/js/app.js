// GNOSISAGENT - Main Application JavaScript

// Theme Management
function initTheme() {
    const themeToggle = document.getElementById('theme-toggle');
    const currentTheme = localStorage.getItem('theme') || 'light';

    document.documentElement.setAttribute('data-theme', currentTheme);
    updateThemeIcon(currentTheme);

    if (themeToggle) {
        themeToggle.addEventListener('click', () => {
            const newTheme = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            updateThemeIcon(newTheme);
        });
    }
}

function updateThemeIcon(theme) {
    const icon = document.querySelector('#theme-toggle .icon');
    if (icon) {
        icon.textContent = theme === 'dark' ? '☀️' : '🌙';
    }
}

// Connection Status
function updateConnectionStatus() {
    const statusDot = document.querySelector('.status-dot');
    const statusText = document.querySelector('.status-text');

    if (navigator.onLine) {
        statusDot?.classList.add('online');
        statusDot?.classList.remove('offline');
        if (statusText) statusText.textContent = 'Online';

        // Sync queued uploads when back online
        if (window.OfflineStorage) {
            window.OfflineStorage.syncQueued().catch(console.error);
        }
    } else {
        statusDot?.classList.remove('online');
        statusDot?.classList.add('offline');
        if (statusText) statusText.textContent = 'Offline';
    }
}

window.addEventListener('online', updateConnectionStatus);
window.addEventListener('offline', updateConnectionStatus);

// File Upload Handling
function initFileUpload() {
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('fileInput');
    const filePreview = document.getElementById('filePreview');
    const removeFile = document.getElementById('removeFile');

    if (!dropZone || !fileInput) return;

    // Drag and drop
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('drag-over');
    });

    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('drag-over');
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('drag-over');

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            fileInput.files = files;
            showFilePreview(files[0]);
        }
    });

    // File input change
    fileInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            showFilePreview(e.target.files[0]);
        }
    });

    // Remove file
    if (removeFile) {
        removeFile.addEventListener('click', () => {
            fileInput.value = '';
            hideFilePreview();
        });
    }
}

function showFilePreview(file) {
    const filePreview = document.getElementById('filePreview');
    const fileName = document.getElementById('fileName');
    const fileSize = document.getElementById('fileSize');
    const dropZoneContent = document.querySelector('.drop-zone-content');

    if (filePreview && fileName && fileSize) {
        fileName.textContent = file.name;
        fileSize.textContent = formatBytes(file.size);

        dropZoneContent.style.display = 'none';
        filePreview.style.display = 'flex';
    }
}

function hideFilePreview() {
    const filePreview = document.getElementById('filePreview');
    const dropZoneContent = document.querySelector('.drop-zone-content');

    if (filePreview) {
        filePreview.style.display = 'none';
        dropZoneContent.style.display = 'block';
    }
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

// Upload data Submission
function initUploadForm() {
    const form = document.getElementById('upload-form');
    if (!form) return;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(form);
        const fileInput = document.getElementById('fileInput');

        if (!fileInput.files[0]) {
            showToast('Please select a file', 'error');
            return;
        }

        formData.append('file', fileInput.files[0]);

        try {
            showProgress();

            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                showToast('Upload successful!', 'success');

                // Save to IndexedDB
                if (window.OfflineStorage) {
                    await window.OfflineStorage.saveDataset(result);
                }

                // Redirect to dataset page
                setTimeout(() => {
                    window.location.href = `/datasets/${result.id}`;
                }, 1000);
            } else {
                throw new Error(await response.text());
            }
        } catch (error) {
            console.error('Upload error:', error);
            showToast('Upload failed: ' + error.message, 'error');

            // Queue for later if offline
            if (!navigator.onLine && window.OfflineStorage) {
                await window.OfflineStorage.queueUpload(Object.fromEntries(formData));
                showToast('Queued for upload when online', 'info');
            }
        } finally {
            hideProgress();
        }
    });
}

function showProgress() {
    const progress = document.getElementById('uploadProgress');
    const submitBtn = document.getElementById('submitBtn');

    if (progress) {
        progress.style.display = 'block';
        animateProgress();
    }

    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.dataset.originalText = submitBtn.textContent;
        submitBtn.textContent = 'Uploading...';
    }
}

function hideProgress() {
    const progress = document.getElementById('uploadProgress');
    const submitBtn = document.getElementById('submitBtn');

    if (progress) progress.style.display = 'none';

    if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.textContent = submitBtn.dataset.originalText || 'Upload & Analyze';
    }
}

function animateProgress() {
    const fill = document.getElementById('progressFill');
    const percent = document.getElementById('progressPercent');
    const text = document.getElementById('progressText');

    let progress = 0;
    const interval = setInterval(() => {
        progress += Math.random() * 30;
        if (progress > 90) progress = 90;

        if (fill) fill.style.width = progress + '%';
        if (percent) percent.textContent = Math.round(progress) + '%';

        if (progress > 30 && progress < 60 && text) {
            text.textContent = 'Processing...';
        } else if (progress >= 60 && text) {
            text.textContent = 'Analyzing data...';
        }
    }, 500);

    // Store interval ID to clear later
    window.progressInterval = interval;
}

// Toast Notifications
// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    updateConnectionStatus();
    initFileUpload();
    initUploadForm();

    // Handle auto-dismissal for toasts added via HTMX
    document.body.addEventListener('htmx:afterSwap', (event) => {
        if (event.detail.target.id === 'toast-container') {
            const lastToast = event.detail.target.lastElementChild;
            if (lastToast && lastToast.classList.contains('toast')) {
                // Remove duplicates if any
                const message = lastToast.querySelector('.toast-message')?.textContent;
                const toasts = event.detail.target.querySelectorAll('.toast');
                toasts.forEach(t => {
                    if (t !== lastToast && t.querySelector('.toast-message')?.textContent === message) {
                        t.remove();
                    }
                });

                // Auto remove after 5s
                setTimeout(() => {
                    dismissToast(lastToast);
                }, 4700);
            }
        }
    });
});

function dismissToast(toast) {
    if (!toast || !toast.parentElement) return;
    toast.classList.add('removing');
    setTimeout(() => {
        if (toast.parentElement) toast.remove();
    }, 300);
}

// Update showToast to use dismissToast
function showToast(message, type = 'info', title = '') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    // Prevent spam: remove existing toasts with the same message
    const existingToasts = container.querySelectorAll('.toast');
    existingToasts.forEach(t => {
        if (t.querySelector('.toast-message')?.textContent === message) {
            t.remove();
        }
    });

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.setAttribute('role', 'alert');
    
    const icon = getToastIcon(type);
    
    toast.innerHTML = `
        <div class="toast-icon">${icon}</div>
        <div class="toast-content">
            ${title ? `<div class="toast-title">${title}</div>` : ''}
            <div class="toast-message">${message}</div>
        </div>
        <button class="toast-close" onclick="dismissToast(this.parentElement)" aria-label="Close">&times;</button>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        dismissToast(toast);
    }, 4700);
}

function getToastIcon(type) {
    const icons = {
        success: '✅',
        error: '❌',
        warning: '⚠️',
        info: 'ℹ️'
    };
    return icons[type] || icons.info;
}

// Collapsible Sections
function toggleSection(id) {
    const section = document.getElementById(id);
    const toggle = event.currentTarget.querySelector('.toggle-icon');

    if (section) {
        const isVisible = section.style.display !== 'none';
        section.style.display = isVisible ? 'none' : 'block';
        if (toggle) toggle.textContent = isVisible ? '▶' : '▼';
    }
}
