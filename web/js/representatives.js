let representatives = [];

async function loadRepresentatives() {
    const loading = document.getElementById('loading');
    const content = document.getElementById('content');
    const errorDisplay = document.getElementById('error-display');
    
    loading.classList.remove('content-hidden');
    content.classList.add('content-hidden');
    errorDisplay.classList.add('content-hidden');

    try {
        const response = await fetch('/api/representatives');
        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Failed to fetch representatives');
        }

        representatives = data.representatives || [];
        renderRepresentatives();

        loading.classList.add('content-hidden');
        content.classList.remove('content-hidden');

    } catch (error) {
        console.error('Error:', error);
        showError(error.message);
        loading.classList.add('content-hidden');
        content.classList.remove('content-hidden');
    }
}

function renderRepresentatives() {
    const container = document.getElementById('representatives-container');
    const countEl = document.getElementById('rep-count');
    
    countEl.textContent = representatives.length;

    if (representatives.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>No representatives found.</p>
                <p><small>Click "Sync from OpenStates" to fetch your representatives.</small></p>
            </div>
        `;
        return;
    }

    let html = '';
    representatives.forEach(rep => {
        html += `
            <div class="representative-card" id="rep-${rep.id}">
                <div class="rep-actions">
                    <button class="btn-small btn-edit" onclick="toggleEdit(${rep.id})">✏️</button>
                    <button class="btn-small btn-delete" onclick="deleteRepresentative(${rep.id})">🗑️</button>
                </div>
                
                <h3>${rep.name}</h3>
                <p><strong>Title:</strong> 
                    <span class="view-mode">${rep.title}</span>
                    <input class="edit-mode" type="text" data-field="title" value="${rep.title}">
                </p>
                
                ${rep.party ? `<p><strong>Party:</strong> 
                    <span class="view-mode">${rep.party}</span>
                    <input class="edit-mode" type="text" data-field="party" value="${rep.party}">
                </p>` : `<p><strong>Party:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <input class="edit-mode" type="text" data-field="party" value="">
                </p>`}
                
                ${rep.district ? `<p><strong>District:</strong> 
                    <span class="view-mode">${rep.district}</span>
                    <input class="edit-mode" type="text" data-field="district" value="${rep.district}">
                </p>` : `<p><strong>District:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <input class="edit-mode" type="text" data-field="district" value="">
                </p>`}
                
                ${rep.email ? `<p><strong>Email:</strong> 
                    <span class="view-mode"><a href="mailto:${rep.email}">${rep.email}</a></span>
                    <input class="edit-mode" type="email" data-field="email" value="${rep.email}">
                </p>` : `<p><strong>Email:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <input class="edit-mode" type="email" data-field="email" value="">
                </p>`}
                
                ${rep.phone ? `<p><strong>Phone:</strong> 
                    <span class="view-mode"><a href="tel:${rep.phone}">${rep.phone}</a></span>
                    <input class="edit-mode" type="tel" data-field="phone" value="${rep.phone}">
                </p>` : `<p><strong>Phone:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <input class="edit-mode" type="tel" data-field="phone" value="">
                </p>`}
                
                ${rep.website ? `<p><strong>Website:</strong> 
                    <span class="view-mode"><a href="${rep.website}" target="_blank">${rep.website}</a></span>
                    <input class="edit-mode" type="url" data-field="website" value="${rep.website}">
                </p>` : `<p><strong>Website:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <input class="edit-mode" type="url" data-field="website" value="">
                </p>`}
                
                ${rep.office_address ? `<p><strong>Office:</strong> 
                    <span class="view-mode">${rep.office_address}</span>
                    <textarea class="edit-mode" data-field="office_address">${rep.office_address}</textarea>
                </p>` : `<p><strong>Office:</strong> 
                    <span class="view-mode"><em>Not specified</em></span>
                    <textarea class="edit-mode" data-field="office_address"></textarea>
                </p>`}
                
                                 <div class="edit-actions">
                     <button class="btn-save" onclick="saveRepresentative(${rep.id})">✅ Save</button>
                     <button class="btn-cancel" onclick="cancelEdit(${rep.id})">❌ Cancel</button>
                 </div>
                 
                 <div class="rep-updated">
                     <small>Last updated: ${new Date(rep.updated_at).toLocaleDateString()}</small>
                 </div>
            </div>
        `;
    });

    container.innerHTML = html;
}

async function syncRepresentatives() {
    const syncBtn = document.getElementById('sync-btn');
    const syncStatus = document.getElementById('sync-status');
    
    syncBtn.disabled = true;
    syncBtn.textContent = '⏳ Syncing...';
    syncStatus.classList.remove('content-hidden');
    syncStatus.innerHTML = '<p>🔄 Fetching representatives from OpenStates API...</p>';

    try {
        const response = await fetch('/api/representatives', {
            method: 'POST'
        });
        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Failed to sync representatives');
        }

        representatives = data.representatives || [];
        renderRepresentatives();
        
        syncStatus.innerHTML = `<p class="status-success">✅ Successfully synced ${data.count} representatives!</p>`;

    } catch (error) {
        console.error('Sync error:', error);
        syncStatus.innerHTML = `<p class="status-error">❌ Sync failed: ${error.message}</p>`;
    } finally {
        syncBtn.disabled = false;
        syncBtn.textContent = '🔄 Sync from OpenStates';
        setTimeout(() => {
            syncStatus.classList.add('content-hidden');
        }, 5000);
    }
}

function showError(message) {
    document.getElementById('error-message').innerHTML = `
        <p>${message}</p>
        <p><small>Make sure USER_ZIP_CODE and OPENSTATES_API_KEY are configured in your .env file.</small></p>
    `;
    document.getElementById('error-display').classList.remove('content-hidden');
}

// Representative editing functions
function toggleEdit(repId) {
    const card = document.getElementById(`rep-${repId}`);
    const viewModes = card.querySelectorAll('.view-mode');
    const editModes = card.querySelectorAll('.edit-mode');
    const editActions = card.querySelector('.edit-actions');
    const editBtn = card.querySelector('.rep-actions button:first-child');
    
    const isInEditMode = editActions.classList.contains('active');
    
    if (isInEditMode) {
        // Switch to view mode
        viewModes.forEach(el => el.classList.remove('hidden'));
        editModes.forEach(el => el.classList.remove('active'));
        editActions.classList.remove('active');
        editBtn.textContent = '✏️';
        editBtn.className = 'btn-small btn-edit';
    } else {
        // Switch to edit mode
        viewModes.forEach(el => el.classList.add('hidden'));
        editModes.forEach(el => el.classList.add('active'));
        editActions.classList.add('active');
        editBtn.textContent = '×';
        editBtn.className = 'btn-small btn-cancel';
    }
}

function cancelEdit(repId) {
    const card = document.getElementById(`rep-${repId}`);
    const rep = representatives.find(r => r.id === repId);
    
    if (rep) {
        // Reset all input values to original data
        const inputs = card.querySelectorAll('.edit-mode');
        inputs.forEach(input => {
            const field = input.dataset.field;
            input.value = rep[field] || '';
        });
    }
    
    // Switch back to view mode
    toggleEdit(repId);
}

async function saveRepresentative(repId) {
    const card = document.getElementById(`rep-${repId}`);
    const inputs = card.querySelectorAll('.edit-mode');
    
    // Collect updated data
    const updates = {};
    inputs.forEach(input => {
        const field = input.dataset.field;
        const value = input.value.trim();
        updates[field] = value || null;
    });

    try {
        // Show loading state
        const saveBtn = card.querySelector('.btn-save');
        const originalText = saveBtn.textContent;
        saveBtn.disabled = true;
        saveBtn.textContent = '⏳ Saving...';

        const response = await fetch(`/api/representatives/${repId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(updates)
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Failed to update representative');
        }

        // Update local data
        const repIndex = representatives.findIndex(r => r.id === repId);
        if (repIndex !== -1) {
            representatives[repIndex] = { ...representatives[repIndex], ...data };
        }

        // Show success and re-render
        showNotification('Representative updated successfully!', 'success');
        renderRepresentatives();

    } catch (error) {
        console.error('Save error:', error);
        showNotification(`Failed to save: ${error.message}`, 'error');
        
        // Reset button state
        const saveBtn = card.querySelector('.btn-save');
        if (saveBtn) {
            saveBtn.disabled = false;
            saveBtn.textContent = '✅ Save Changes';
        }
    }
}

async function deleteRepresentative(repId) {
    const rep = representatives.find(r => r.id === repId);
    if (!rep) return;

    if (!confirm(`Are you sure you want to delete ${rep.name}? This action cannot be undone.`)) {
        return;
    }

    try {
        const response = await fetch(`/api/representatives/${repId}`, {
            method: 'DELETE'
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Failed to delete representative');
        }

        // Remove from local data and re-render
        representatives = representatives.filter(r => r.id !== repId);
        renderRepresentatives();
        showNotification('Representative deleted successfully!', 'success');

    } catch (error) {
        console.error('Delete error:', error);
        showNotification(`Failed to delete: ${error.message}`, 'error');
    }
}

function showNotification(message, type = 'success') {
    // Remove existing notifications
    const existing = document.querySelector('.notification');
    if (existing) {
        existing.remove();
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;

    document.body.appendChild(notification);

    // Auto remove after 5 seconds
    setTimeout(() => {
        if (notification.parentNode) {
            notification.classList.add('slide-out');
            setTimeout(() => notification.remove(), 300);
        }
    }, 5000);
}

// Event listeners and initialization
document.addEventListener('DOMContentLoaded', function() {
    // Load data when page loads
    loadRepresentatives();
    
    // Set up event listeners
    document.getElementById('sync-btn').addEventListener('click', syncRepresentatives);
    document.getElementById('refresh-btn').addEventListener('click', loadRepresentatives);
}); 