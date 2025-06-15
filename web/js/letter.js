document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('letter-gen-form');
    
    if (form) {
        form.addEventListener('submit', handleFormSubmit);
    }
});

function handleFormSubmit(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const advocacy = {
        main_issue: formData.get('main-issue')?.trim(),
        specific_concern: formData.get('specific-concern')?.trim(),
        requested_action: formData.get('requested-action')?.trim()
    };
    
    // Validate required fields
    if (!advocacy.main_issue || !advocacy.specific_concern || !advocacy.requested_action) {
        showError('Please fill in all required fields');
        return;
    }
    
    const button = e.target.querySelector('button[type="submit"]');
    setButtonLoading(button, true);
    
    // Add 3-second cooldown to prevent rapid requests
    setTimeout(() => {
        generateLetter(advocacy, button);
    }, 3000);
}

function generateLetter(advocacy, button) {
    const requestData = {
        advocacy: advocacy
    };
    
    fetch('/api/letters/generate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestData)
    })
    .then(response => response.json())
    .then(data => {
        setButtonLoading(button, false);
        
        if (data.error) {
            showError(data.error);
        } else {
            showResult(data);
        }
    })
    .catch(error => {
        setButtonLoading(button, false);
        console.error('Error:', error);
        showError('Failed to generate letter. Please try again.');
    });
}

function showResult(data) {
    const container = document.getElementById('result-container');
    
    container.innerHTML = `
        <div class="result-header">
            <h3>✅ Letter Generated Successfully!</h3>
            <div class="info-message">
                <p><strong>Configuration Used:</strong> ${data.configuration_used.max_length} words max, ${data.configuration_used.tone} tone, ${data.configuration_used.ai_provider} (${data.configuration_used.ai_model})</p>
            </div>
        </div>
        
        <div class="ai-selection-info">
            <h4>🤖 AI Representative Selection</h4>
            <div class="selected-rep-card">
                <strong>${data.letter.selected_representative.title} ${data.letter.selected_representative.name}</strong>
                <span class="rep-details">${data.letter.selected_representative.state}${data.letter.selected_representative.party ? ` - ${data.letter.selected_representative.party}` : ''}</span>
                <div class="selection-reasoning">
                    <em>${data.ai_selection.reasoning}</em>
                </div>
            </div>
        </div>

        <div class="letter-header">
            <h4>📝 Generated Letter</h4>
        </div>
        
        <div class="letter-body">
            <pre>${data.letter.content}</pre>
        </div>
        
        <div class="letter-actions">
            <button class="btn btn-primary" onclick="copyToClipboard()">📋 Copy Letter</button>
            <button class="btn btn-secondary" onclick="downloadLetter()">💾 Download as Text</button>
        </div>
        
        <div class="letter-metadata">
            <small>
                Generated: ${new Date(data.letter.created_at).toLocaleString()} | 
                Tokens: ${data.letter.metadata.tokens_used} | 
                Actual Length: ~${data.letter.content.split(' ').length} words
            </small>
        </div>
    `;
    
    container.classList.remove('hidden');
    
    // Store letter content for copy/download functions
    window.currentLetter = data.letter.content;
}

function showError(message) {
    const errorHtml = `
        <div class="error-message">
            <h3>❌ Error</h3>
            <p>${message}</p>
            <button onclick="location.reload()" class="btn btn-primary">Try Again</button>
        </div>
    `;
    
    // Show error below form or replace content
    const container = document.getElementById('letter-gen-content');
    const existingError = container.querySelector('.error-message');
    if (existingError) {
        existingError.remove();
    }
    
    container.insertAdjacentHTML('beforeend', errorHtml);
}

function setButtonLoading(button, isLoading) {
    if (isLoading) {
        button.disabled = true;
        button.textContent = 'Generating Letter... (please wait 3 seconds)';
        button.classList.add('loading');
    } else {
        button.disabled = false;
        button.textContent = 'Generate Letter';
        button.classList.remove('loading');
    }
}

function copyToClipboard() {
    if (!window.currentLetter) {
        showNotification('No letter content to copy', 'error');
        return;
    }
    
    navigator.clipboard.writeText(window.currentLetter).then(() => {
        showNotification('Letter copied to clipboard!', 'success');
    }).catch(err => {
        console.error('Failed to copy: ', err);
        showNotification('Failed to copy letter', 'error');
    });
}

function downloadLetter() {
    if (!window.currentLetter) {
        showNotification('No letter content to download', 'error');
        return;
    }
    
    const blob = new Blob([window.currentLetter], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `advocacy-letter-${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    showNotification('Letter downloaded!', 'success');
}

function generateNewLetter() {
    location.reload();
}

function showNotification(message, type = 'success') {
    // Remove any existing notifications
    const existing = document.querySelector('.floating-notification');
    if (existing) {
        existing.remove();
    }
    
    const notification = document.createElement('div');
    notification.className = `floating-notification ${type}`;
    notification.textContent = message;
    
    document.body.appendChild(notification);
    
    // Remove after 3 seconds
    setTimeout(() => {
        if (notification.parentNode) {
            notification.classList.add('slide-up');
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.remove();
                }
            }, 300);
        }
    }, 3000);
}
