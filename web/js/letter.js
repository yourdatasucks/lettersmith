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
    const selectedRep = data.letter.selected_representative;
    const letter = data.letter;
    const metadata = letter.metadata;
    
    const resultHtml = `
        <div class="letter-result">
            <div class="result-header">
                <h3>✅ Letter Generated Successfully!</h3>
                <div class="ai-selection-info">
                    <h4>🤖 AI Selected Representative:</h4>
                    <div class="selected-rep-card">
                        <strong>${selectedRep.title} ${selectedRep.name}</strong>
                        <span class="rep-details">
                            ${selectedRep.state}${selectedRep.party ? ` (${selectedRep.party})` : ''}
                            ${selectedRep.district ? ` - District ${selectedRep.district}` : ''}
                        </span>
                        <p class="selection-reasoning">
                            ${data.ai_selection.reasoning}
                        </p>
                    </div>
                </div>
            </div>
            
            <div class="letter-content">
                <div class="letter-header">
                    <h4>Subject: ${letter.subject}</h4>
                </div>
                
                <div class="letter-body">
                    <pre>${letter.content}</pre>
                </div>
            </div>
            
            <div class="letter-actions">
                <button onclick="copyToClipboard('${letter.content.replace(/'/g, "\\'")}', this)" class="btn btn-secondary">
                    Copy Letter
                </button>
                <button onclick="generateNewLetter()" class="btn btn-primary">
                    Generate Another Letter
                </button>
            </div>
            
            <div class="letter-metadata">
                <small>
                    Generated using ${metadata.provider} (${metadata.model}) • 
                    ${metadata.tokens_used} tokens • 
                    ${metadata.tone} tone • 
                    Theme: ${metadata.theme}
                </small>
            </div>
        </div>
    `;
    
    // Replace form with result
    const container = document.getElementById('letter-gen-content');
    container.innerHTML = resultHtml;
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

function copyToClipboard(text, button) {
    navigator.clipboard.writeText(text).then(() => {
        const originalText = button.textContent;
        button.textContent = '✅ Copied!';
        button.classList.add('success');
        
        setTimeout(() => {
            button.textContent = originalText;
            button.classList.remove('success');
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy text: ', err);
        button.textContent = '❌ Copy Failed';
        button.classList.add('error');
        
        setTimeout(() => {
            button.textContent = 'Copy Letter';
            button.classList.remove('error');
        }, 2000);
    });
}

function generateNewLetter() {
    location.reload();
}
