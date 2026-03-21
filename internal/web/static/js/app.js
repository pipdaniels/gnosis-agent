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

// Form Submission
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
        submitBtn.querySelector('.btn-text').style.display = 'none';
        submitBtn.querySelector('.btn-loader').style.display = 'inline';
    }
}

function hideProgress() {
    const progress = document.getElementById('uploadProgress');
    const submitBtn = document.getElementById('submitBtn');

    if (progress) progress.style.display = 'none';

    if (submitBtn) {
        submitBtn.disabled = false;
        submitBtn.querySelector('.btn-text').style.display = 'inline';
        submitBtn.querySelector('.btn-loader').style.display = 'none';
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
function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container') || createToastContainer();

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
		<span class="toast-icon">${getToastIcon(type)}</span>
		<span class="toast-message">${message}</span>
		<button class="toast-close" onclick="this.parentElement.remove()">✕</button>
	`;

    container.appendChild(toast);

    setTimeout(() => toast.remove(), 5000);
}

function createToastContainer() {
    const container = document.createElement('div');
    container.id = 'toast-container';
    container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
    document.body.appendChild(container);
    return container;
}

function getToastIcon(type) {
    const icons = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
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

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    updateConnectionStatus();
    initFileUpload();
    initUploadForm();
});
