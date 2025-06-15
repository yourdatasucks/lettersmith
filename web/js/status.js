let statusData = null;
let previousStatusData = null;

async function fetchSystemStatus() {
    try {
        const response = await fetch('/api/system/status');
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return await response.json();
    } catch (error) {
        console.error('Failed to fetch system status:', error);
        throw error;
    }
}

function getStatusBadgeClass(status) {
    const statusMap = {
        'healthy': 'status-healthy',
        'error': 'status-error',
        'not_configured': 'status-not-configured',
        'not_implemented': 'status-not-implemented',
        'incomplete': 'status-incomplete',
        'misconfigured': 'status-misconfigured'
    };
    return statusMap[status] || 'status-not-configured';
}

function getStatusText(status) {
    const statusMap = {
        'healthy': 'Healthy',
        'error': 'Error',
        'not_configured': 'Not Configured',
        'not_implemented': 'Not Implemented',
        'incomplete': 'Incomplete',
        'misconfigured': 'Misconfigured'
    };
    return statusMap[status] || status;
}

function hasServiceChanged(serviceKey, newService, oldServices) {
    if (!oldServices || !oldServices[serviceKey]) {
        return true; // New service
    }
    
    const oldService = oldServices[serviceKey];
    return (
        oldService.status !== newService.status ||
        oldService.details !== newService.details ||
        oldService.name !== newService.name
    );
}

function updateServiceCard(serviceKey, service) {
    let card = document.querySelector(`[data-service="${serviceKey}"]`);
    
    if (!card) {
        // Create new card
        card = document.createElement('div');
        card.className = 'service-card';
        card.setAttribute('data-service', serviceKey);
        document.getElementById('services-grid').appendChild(card);
    }
    
    // Update card content
    card.innerHTML = `
        <div class="service-header">
            <span class="service-name">${service.name}</span>
            <span class="status-badge ${getStatusBadgeClass(service.status)}">
                ${getStatusText(service.status)}
            </span>
        </div>
        <div class="service-details">${service.details}</div>
    `;
    
    // Add subtle animation for changes
    card.classList.add('service-card-updating');
    setTimeout(() => {
        card.classList.remove('service-card-updating');
    }, 200);
}

function renderServices(services, isInitialLoad = false) {
    const grid = document.getElementById('services-grid');
    
    if (isInitialLoad) {
        // Clear grid on initial load
        grid.innerHTML = '';
    }
    
    // Update or create service cards
    Object.entries(services).forEach(([key, service]) => {
        if (isInitialLoad || hasServiceChanged(key, service, previousStatusData?.services)) {
            updateServiceCard(key, service);
        }
    });
    
    // Remove cards for services that no longer exist
    if (!isInitialLoad && previousStatusData?.services) {
        Object.keys(previousStatusData.services).forEach(oldKey => {
            if (!services[oldKey]) {
                const oldCard = document.querySelector(`[data-service="${oldKey}"]`);
                if (oldCard) {
                    oldCard.remove();
                }
            }
        });
    }
}

function hasOverallStatusChanged(newStatus, newSummary) {
    if (!previousStatusData) return true;
    
    return (
        previousStatusData.overall_status !== newStatus.overall_status ||
        previousStatusData.summary.healthy_services !== newSummary.healthy_services ||
        previousStatusData.summary.total_services !== newSummary.total_services ||
        previousStatusData.summary.completion_percentage !== newSummary.completion_percentage
    );
}

function renderOverallStatus(status, summary, isInitialLoad = false) {
    if (!isInitialLoad && !hasOverallStatusChanged(status, summary)) {
        // Only update timestamp if nothing else changed
        const lastUpdated = document.getElementById('last-updated');
        lastUpdated.textContent = `Last updated: ${new Date(status.timestamp).toLocaleString()}`;
        return;
    }
    
    const overallDiv = document.getElementById('overall-status');
    const title = document.getElementById('overall-title');
    const summaryEl = document.getElementById('overall-summary');
    const completionFill = document.getElementById('completion-fill');
    const lastUpdated = document.getElementById('last-updated');

    // Update overall status class with smooth transition
    overallDiv.className = `overall-status ${status.overall_status}`;
    
    // Update title
    const statusTitles = {
        'healthy': '✅ All Systems Operational',
        'incomplete': '⚠️ System Setup Incomplete',
        'degraded': '❌ System Issues Detected'
    };
    title.textContent = statusTitles[status.overall_status] || 'System Status';

    // Smooth progress bar animation
    const percentage = summary.completion_percentage || 0;
    completionFill.style.width = `${percentage}%`;
    
    summaryEl.innerHTML = `
        <strong>${summary.healthy_services} of ${summary.total_services}</strong> services healthy
        <br>
        <span class="status-summary-detail">System ${percentage}% configured</span>
    `;

    // Update timestamp
    lastUpdated.textContent = `Last updated: ${new Date(status.timestamp).toLocaleString()}`;
}

function hasMissingComponentsChanged(newMissing) {
    if (!previousStatusData) return true;
    
    const oldMissing = previousStatusData.missing_components || [];
    
    if (oldMissing.length !== newMissing.length) return true;
    
    return !oldMissing.every((item, index) => item === newMissing[index]);
}

function renderMissingComponents(missing, isInitialLoad = false) {
    if (!isInitialLoad && !hasMissingComponentsChanged(missing)) {
        return; // No changes needed
    }
    
    const container = document.getElementById('missing-components');
    const list = document.getElementById('missing-list');

    if (missing.length === 0) {
        container.classList.add('content-hidden');
        return;
    }

    container.classList.remove('content-hidden');
    list.innerHTML = '';

    missing.forEach(component => {
        const li = document.createElement('li');
        li.textContent = component;
        list.appendChild(li);
    });
}

function showError(message) {
    const loadingIndicator = document.getElementById('loading-indicator');
    loadingIndicator.innerHTML = `
        <div class="error-display">
            <h3>Error Loading Status</h3>
            <p>${message}</p>
            <button onclick="updateStatus(true)" class="refresh-btn">Retry</button>
        </div>
    `;
    loadingIndicator.classList.remove('content-hidden');
}

function hideLoading() {
    const loadingIndicator = document.getElementById('loading-indicator');
    const statusContent = document.getElementById('status-content');
    
    loadingIndicator.classList.add('content-hidden');
    statusContent.classList.remove('content-hidden');
}

function showLoading() {
    const loadingIndicator = document.getElementById('loading-indicator');
    const statusContent = document.getElementById('status-content');
    
    loadingIndicator.innerHTML = `
        <div class="status-loading"></div>
        <p>Checking system status...</p>
    `;
    loadingIndicator.classList.remove('content-hidden');
    statusContent.classList.add('content-hidden');
}

async function updateStatus(isInitialLoad = false) {
    const refreshBtn = document.getElementById('refresh-btn');

    try {
        // Only show loading spinner on initial load or manual refresh
        if (isInitialLoad) {
            showLoading();
        }
        
        if (refreshBtn) {
            refreshBtn.disabled = true;
        }

        const newStatusData = await fetchSystemStatus();

        // Store previous data for comparison
        previousStatusData = statusData;
        statusData = newStatusData;

        // Update components (they'll only re-render if changed)
        renderOverallStatus(statusData, statusData.summary, isInitialLoad);
        renderServices(statusData.services, isInitialLoad);
        renderMissingComponents(statusData.missing_components, isInitialLoad);

        // Hide loading on initial load
        if (isInitialLoad) {
            hideLoading();
        }

    } catch (error) {
        console.error('Status update failed:', error);
        
        if (isInitialLoad) {
            showError(error.message);
        } else {
            // For background updates, just log the error and keep showing old data
            console.warn('Background status update failed, keeping existing data');
        }
    } finally {
        if (refreshBtn) {
            refreshBtn.disabled = false;
        }
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    updateStatus(true); // Initial load with loading spinner
    
    // Add refresh button handler
    const refreshBtn = document.getElementById('refresh-btn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', () => updateStatus(true)); // Manual refresh with spinner
    }
});

// Auto-refresh every 30 seconds (background updates without spinner)
setInterval(() => updateStatus(false), 30000); 