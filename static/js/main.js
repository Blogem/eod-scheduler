// EoD Scheduler - Interactive Features

// Theme Management
class ThemeManager {
    constructor() {
        this.currentTheme = localStorage.getItem('eod-theme') || 'neutral';
        this.init();
    }

    init() {
        this.applyTheme(this.currentTheme);
        this.createThemeToggle();
    }

    applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        this.currentTheme = theme;
        localStorage.setItem('eod-theme', theme);
    }

    toggleTheme() {
        const newTheme = this.currentTheme === 'neutral' ? 'heineken' : 'neutral';
        this.applyTheme(newTheme);
        this.updateToggleButton();
    }

    createThemeToggle() {
        const button = document.querySelector('.theme-toggle');
        console.log('Looking for theme toggle button:', button);
        if (button) {
            this.updateToggleButton(button);
            button.addEventListener('click', () => this.toggleTheme());
            console.log('Theme toggle functionality added');
        } else {
            console.log('Theme toggle button not found');
        }
    }

    updateToggleButton(button = document.querySelector('.theme-toggle')) {
        if (button) {
            if (this.currentTheme === 'heineken') {
                button.innerHTML = 'Heineken';
                button.title = 'Switch to neutral theme';
            } else {
                button.innerHTML = 'Neutral';
                button.title = 'Switch to Heineken theme';
            }
        }
    }
}

document.addEventListener('DOMContentLoaded', function () {
    // Initialize theme management
    const themeManager = new ThemeManager();
    // Add loading states to form submissions
    const forms = document.querySelectorAll('form');
    forms.forEach(form => {
        form.addEventListener('submit', function (e) {
            const submitBtn = form.querySelector('button[type="submit"], input[type="submit"]');
            if (submitBtn) {
                submitBtn.classList.add('btn-loading');
                submitBtn.disabled = true;

                // Re-enable button after 3 seconds as fallback
                setTimeout(() => {
                    submitBtn.classList.remove('btn-loading');
                    submitBtn.disabled = false;
                }, 3000);
            }
        });
    });

    // Add confirmation dialogs for delete actions
    const deleteButtons = document.querySelectorAll('.btn-danger[data-confirm]');
    deleteButtons.forEach(btn => {
        btn.addEventListener('click', function (e) {
            const message = this.getAttribute('data-confirm') || 'Are you sure you want to delete this item?';
            if (!confirm(message)) {
                e.preventDefault();
                return false;
            }
        });
    });

    // Auto-hide success/error messages after 5 seconds
    const messages = document.querySelectorAll('.message');
    messages.forEach(message => {
        if (message.classList.contains('message-success') || message.classList.contains('message-error')) {
            setTimeout(() => {
                message.style.opacity = '0';
                message.style.transform = 'translateY(-10px)';
                setTimeout(() => {
                    message.remove();
                }, 300);
            }, 5000);
        }
    });

    // Add smooth scrolling for anchor links
    const anchorLinks = document.querySelectorAll('a[href^="#"]');
    anchorLinks.forEach(link => {
        link.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });

    // Add keyboard navigation for tables
    const tables = document.querySelectorAll('table');
    tables.forEach(table => {
        table.setAttribute('tabindex', '0');
        table.addEventListener('keydown', function (e) {
            if (e.key === 'Enter' || e.key === ' ') {
                const focusedRow = this.querySelector('tr:focus');
                if (focusedRow) {
                    const firstLink = focusedRow.querySelector('a');
                    if (firstLink) {
                        firstLink.click();
                    }
                }
            }
        });
    });

    // Add focus management for modal-like interactions
    const overlayTriggers = document.querySelectorAll('[data-toggle="overlay"]');
    overlayTriggers.forEach(trigger => {
        trigger.addEventListener('click', function () {
            const targetId = this.getAttribute('data-target');
            const target = document.querySelector(targetId);
            if (target) {
                target.style.display = 'block';
                target.focus();

                // Close on Escape key
                const closeHandler = (e) => {
                    if (e.key === 'Escape') {
                        target.style.display = 'none';
                        trigger.focus();
                        document.removeEventListener('keydown', closeHandler);
                    }
                };
                document.addEventListener('keydown', closeHandler);
            }
        });
    });

    // Enhanced form validation feedback
    const inputs = document.querySelectorAll('input, select, textarea');
    inputs.forEach(input => {
        input.addEventListener('blur', function () {
            if (this.checkValidity()) {
                this.classList.remove('invalid');
                this.classList.add('valid');
            } else {
                this.classList.remove('valid');
                this.classList.add('invalid');
            }
        });

        input.addEventListener('input', function () {
            if (this.classList.contains('invalid') && this.checkValidity()) {
                this.classList.remove('invalid');
                this.classList.add('valid');
            }
        });
    });

    // Add current time display
    const currentTimeElements = document.querySelectorAll('.current-time');
    if (currentTimeElements.length > 0) {
        function updateTime() {
            const now = new Date();
            const timeString = now.toLocaleTimeString();
            currentTimeElements.forEach(el => {
                el.textContent = timeString;
            });
        }
        updateTime();
        setInterval(updateTime, 1000);
    }

    // Add tooltips for help text
    const helpElements = document.querySelectorAll('[data-help]');
    helpElements.forEach(element => {
        element.setAttribute('title', element.getAttribute('data-help'));
        element.style.cursor = 'help';
    });

    // Add copy-to-clipboard functionality
    const copyButtons = document.querySelectorAll('[data-copy]');
    copyButtons.forEach(button => {
        button.addEventListener('click', function () {
            const textToCopy = this.getAttribute('data-copy');
            if (navigator.clipboard) {
                navigator.clipboard.writeText(textToCopy).then(() => {
                    // Show feedback
                    const originalText = this.textContent;
                    this.textContent = 'Copied!';
                    setTimeout(() => {
                        this.textContent = originalText;
                    }, 2000);
                });
            }
        });
    });

    // Improve schedule grid interactions
    const scheduleEntries = document.querySelectorAll('.schedule-entry');
    scheduleEntries.forEach(entry => {
        entry.addEventListener('mouseenter', function () {
            this.style.transform = 'translateY(-2px) scale(1.02)';
        });

        entry.addEventListener('mouseleave', function () {
            this.style.transform = '';
        });
    });

    // Add print functionality
    const printButtons = document.querySelectorAll('.btn-print');
    printButtons.forEach(button => {
        button.addEventListener('click', function () {
            window.print();
        });
    });

    // Dark mode toggle (basic implementation)
    const darkModeToggle = document.querySelector('.dark-mode-toggle');
    if (darkModeToggle) {
        darkModeToggle.addEventListener('click', function () {
            document.body.classList.toggle('dark-mode');
            localStorage.setItem('darkMode', document.body.classList.contains('dark-mode'));
        });

        // Load saved dark mode preference
        if (localStorage.getItem('darkMode') === 'true') {
            document.body.classList.add('dark-mode');
        }
    }
});

// Utility functions
window.EODScheduler = {
    showMessage: function (message, type = 'info') {
        const messageEl = document.createElement('div');
        messageEl.className = `message message-${type}`;
        messageEl.textContent = message;

        const container = document.querySelector('main .container');
        if (container) {
            container.insertBefore(messageEl, container.firstChild);

            // Auto-hide after 5 seconds
            setTimeout(() => {
                messageEl.style.opacity = '0';
                messageEl.style.transform = 'translateY(-10px)';
                setTimeout(() => messageEl.remove(), 300);
            }, 5000);
        }
    },

    confirmAction: function (message, callback) {
        if (confirm(message)) {
            callback();
        }
    },

    formatTime: function (timeString) {
        const time = new Date(`2000-01-01T${timeString}`);
        return time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    },

    formatDate: function (dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString();
    }
};