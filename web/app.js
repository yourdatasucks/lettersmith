document.addEventListener('DOMContentLoaded', function() {
    
    loadConfiguration();

    
    document.getElementById('ai-provider').addEventListener('change', handleAIProviderChange);
    document.getElementById('email-provider').addEventListener('change', handleEmailProviderChange);
    document.getElementById('generation-method').addEventListener('change', handleGenerationMethodChange);
    document.getElementById('smtp-preset').addEventListener('change', handleSMTPPresetChange);
    document.getElementById('save-config').addEventListener('click', saveConfiguration);
    document.getElementById('test-config').addEventListener('click', testConfiguration);
    
    
    setupRealTimeTrimming();
});


function setupRealTimeTrimming() {
    
    const inputSelectors = [
        'input[type="text"]',
        'input[type="email"]', 
        'input[type="password"]',
        'textarea'
    ];
    
    const inputs = document.querySelectorAll(inputSelectors.join(', '));
    
    inputs.forEach(input => {
        
        input.addEventListener('paste', function(e) {
            
            setTimeout(() => {
                this.value = this.value.trim();
            }, 10);
        });
        
        
        input.addEventListener('blur', function(e) {
            this.value = this.value.trim();
        });
    });
}


function handleAIProviderChange(e) {
    const provider = e.target.value;
    
    
    document.getElementById('openai-config').classList.add('hidden');
    document.getElementById('anthropic-config').classList.add('hidden');
    
    
    if (provider === 'openai') {
        document.getElementById('openai-config').classList.remove('hidden');
    } else if (provider === 'anthropic') {
        document.getElementById('anthropic-config').classList.remove('hidden');
    }
}

function handleEmailProviderChange(e) {
    const provider = e.target.value;
    
    
    document.getElementById('smtp-config').classList.add('hidden');
    document.getElementById('sendgrid-config').classList.add('hidden');
    document.getElementById('mailgun-config').classList.add('hidden');
    
    
    if (provider === 'smtp') {
        document.getElementById('smtp-config').classList.remove('hidden');
    } else if (provider === 'sendgrid') {
        document.getElementById('sendgrid-config').classList.remove('hidden');
    } else if (provider === 'mailgun') {
        document.getElementById('mailgun-config').classList.remove('hidden');
    }
}

function handleGenerationMethodChange(e) {
    const method = e.target.value;
    const templateConfig = document.getElementById('template-config');
    const aiSection = document.getElementById('ai-section');
    
    if (method === 'templates') {
        templateConfig.classList.remove('hidden');
        
        aiSection.classList.add('hidden');
    } else {
        templateConfig.classList.add('hidden');
        
        aiSection.classList.remove('hidden');
        
        
        const aiProvider = document.getElementById('ai-provider').value;
        handleAIProviderChange({ target: { value: aiProvider } });
    }
}

function handleSMTPPresetChange(e) {
    const preset = e.target.value;
    const hostField = document.getElementById('smtp-host');
    const portField = document.getElementById('smtp-port');
    const hostHelp = document.getElementById('smtp-host-help');
    const portHelp = document.getElementById('smtp-port-help');
    const usernameHelp = document.getElementById('smtp-username-help');
    const passwordHelp = document.getElementById('smtp-password-help');
    const setupInstructions = document.getElementById('setup-instructions');
    
    switch (preset) {
        case 'protonmail':
            hostField.value = '127.0.0.1';
            portField.value = '1025';
            hostHelp.textContent = 'ProtonMail Bridge running on your host machine';
            portHelp.textContent = 'Default Bridge port (may vary, check Bridge app)';
            usernameHelp.textContent = 'Your ProtonMail email address';
            passwordHelp.textContent = 'Use the Bridge password from ProtonMail Bridge app';
            setupInstructions.innerHTML = `
                <p><strong>ProtonMail Bridge Setup:</strong></p>
                <ol>
                    <li>Download ProtonMail Bridge from <a href="https://proton.me/mail/bridge" target="_blank">proton.me/mail/bridge</a></li>
                    <li>Install and log in to Bridge with your ProtonMail account</li>
                    <li>Copy the Bridge password from the Bridge app (NOT your ProtonMail password)</li>
                    <li>Use your full ProtonMail email as username</li>
                </ol>
                <p><em>💡 Tip: Bridge must be running for email sending to work</em></p>
            `;
            break;
            
        case 'gmail':
            hostField.value = 'smtp.gmail.com';
            portField.value = '587';
            hostHelp.textContent = 'Gmail SMTP server';
            portHelp.textContent = 'Standard SMTP port with STARTTLS';
            usernameHelp.textContent = 'Your Gmail address';
            passwordHelp.textContent = 'Use an App Password (NOT your regular Gmail password)';
            setupInstructions.innerHTML = `
                <p><strong>Gmail Setup:</strong></p>
                <ol>
                    <li>Enable 2-Factor Authentication on your Google account</li>
                    <li>Go to Google Account Settings → Security → App passwords</li>
                    <li>Generate a new app password for "Mail"</li>
                    <li>Use your Gmail address as username</li>
                    <li>Use the generated app password (16 characters)</li>
                </ol>
                <p><em>💡 Tip: Regular Gmail passwords won't work - you must use an App Password</em></p>
            `;
            break;
            
        case 'outlook':
            hostField.value = 'smtp-mail.outlook.com';
            portField.value = '587';
            hostHelp.textContent = 'Outlook/Hotmail SMTP server';
            portHelp.textContent = 'Standard SMTP port with STARTTLS';
            usernameHelp.textContent = 'Your Outlook/Hotmail email address';
            passwordHelp.textContent = 'Your Outlook password or app password if 2FA enabled';
            setupInstructions.innerHTML = `
                <p><strong>Outlook/Hotmail Setup:</strong></p>
                <ol>
                    <li>Use your full Outlook/Hotmail email address as username</li>
                    <li>If you have 2FA enabled, generate an app password</li>
                    <li>Otherwise, use your regular Outlook password</li>
                </ol>
                <p><em>💡 Tip: If you get authentication errors, try enabling "Less secure app access" or use an app password</em></p>
            `;
            break;
            
        case 'yahoo':
            hostField.value = 'smtp.mail.yahoo.com';
            portField.value = '587';
            hostHelp.textContent = 'Yahoo Mail SMTP server';
            portHelp.textContent = 'Standard SMTP port with STARTTLS';
            usernameHelp.textContent = 'Your Yahoo email address';
            passwordHelp.textContent = 'Use an App Password (required for Yahoo)';
            setupInstructions.innerHTML = `
                <p><strong>Yahoo Mail Setup:</strong></p>
                <ol>
                    <li>Go to Yahoo Account Settings → Security</li>
                    <li>Generate an app password for "Desktop app"</li>
                    <li>Use your Yahoo email address as username</li>
                    <li>Use the generated app password (NOT your Yahoo password)</li>
                </ol>
                <p><em>💡 Tip: Yahoo requires app passwords for SMTP access</em></p>
            `;
            break;
            
        default:
            hostHelp.textContent = 'Enter your SMTP server hostname or IP';
            portHelp.textContent = 'Common ports: 25, 465 (SSL), 587 (STARTTLS)';
            usernameHelp.textContent = 'Usually your email address';
            passwordHelp.textContent = 'Your email password or app password';
            setupInstructions.innerHTML = `
                <p><strong>Custom SMTP Setup:</strong></p>
                <ol>
                    <li>Contact your email provider for SMTP settings</li>
                    <li>Common ports: 25 (unencrypted), 465 (SSL), 587 (STARTTLS)</li>
                    <li>Use your full email address as username</li>
                    <li>Use your email password or app password if 2FA is enabled</li>
                </ol>
            `;
            break;
    }
}


async function loadConfiguration() {
    try {
        const response = await fetch('/api/config');
        if (!response.ok) throw new Error('Failed to load configuration');
        
        const config = await response.json();
        
        
        const envValues = config.env_values || {};
        
        
        
        document.getElementById('user-name').value = envValues.USER_NAME || config.user?.Name || '';
        document.getElementById('user-email').value = envValues.USER_EMAIL || config.user?.Email || '';
        document.getElementById('user-zip').value = envValues.USER_ZIP_CODE || config.user?.ZipCode || '';
        
        
        const sendCopyValue = envValues.SEND_COPY_TO_SELF;
        if (sendCopyValue !== undefined) {
            document.getElementById('send-copy-to-self').checked = sendCopyValue === 'true';
        } else {
            document.getElementById('send-copy-to-self').checked = config.user?.SendCopyToSelf !== false;
        }
        
        
        if (config.letter) {
            const generationMethod = envValues.LETTER_GENERATION_METHOD || config.letter.GenerationMethod || 'ai';
            document.getElementById('generation-method').value = generationMethod;
            handleGenerationMethodChange({ target: { value: generationMethod } });
            
            document.getElementById('letter-tone').value = envValues.LETTER_TONE || config.letter.Tone || 'professional';
            document.getElementById('max-length').value = envValues.LETTER_MAX_LENGTH || config.letter.MaxLength || 500;
            
            
            if (config.letter.TemplateConfig) {
                document.getElementById('template-directory').value = envValues.TEMPLATE_DIRECTORY || config.letter.TemplateConfig.Directory || 'templates/';
                document.getElementById('rotation-strategy').value = envValues.TEMPLATE_ROTATION_STRATEGY || config.letter.TemplateConfig.RotationStrategy || 'random-unique';
                
                
                const templatePersonalize = envValues.TEMPLATE_PERSONALIZE;
                if (templatePersonalize !== undefined) {
                    document.getElementById('personalize-templates').checked = templatePersonalize === 'true';
                } else {
                    document.getElementById('personalize-templates').checked = config.letter.TemplateConfig.Personalize !== false;
                }
            } else {
                
                if (envValues.TEMPLATE_DIRECTORY || envValues.TEMPLATE_ROTATION_STRATEGY || envValues.TEMPLATE_PERSONALIZE) {
                    document.getElementById('template-directory').value = envValues.TEMPLATE_DIRECTORY || 'templates/';
                    document.getElementById('rotation-strategy').value = envValues.TEMPLATE_ROTATION_STRATEGY || 'random-unique';
                    
                    const templatePersonalize = envValues.TEMPLATE_PERSONALIZE;
                    if (templatePersonalize !== undefined) {
                        document.getElementById('personalize-templates').checked = templatePersonalize === 'true';
                    } else {
                        document.getElementById('personalize-templates').checked = true; 
                    }
                }
            }
        }
        
        
        const aiProvider = envValues.AI_PROVIDER || config.ai?.provider || '';
        document.getElementById('ai-provider').value = aiProvider;
        handleAIProviderChange({ target: { value: aiProvider } });
        
        if (config.ai) {
            if (config.ai.openai) {
                document.getElementById('openai-model').value = envValues.OPENAI_MODEL || config.ai.openai.model || 'gpt-4';
                
                if (envValues.OPENAI_API_KEY || config.ai.openai.configured) {
                    const keyInput = document.getElementById('openai-key');
                    keyInput.placeholder = 'API key configured (leave blank to keep current)';
                    keyInput.classList.add('configured');
                }
            }
            
            if (config.ai.anthropic) {
                document.getElementById('anthropic-model').value = envValues.ANTHROPIC_MODEL || config.ai.anthropic.model || 'claude-3-sonnet-20240229';
                
                if (envValues.ANTHROPIC_API_KEY || config.ai.anthropic.configured) {
                    const keyInput = document.getElementById('anthropic-key');
                    keyInput.placeholder = 'API key configured (leave blank to keep current)';
                    keyInput.classList.add('configured');
                }
            }
        }
        
        
        const emailProvider = envValues.EMAIL_PROVIDER || config.email?.provider || '';
        document.getElementById('email-provider').value = emailProvider;
        handleEmailProviderChange({ target: { value: emailProvider } });
        
        if (config.email) {
            if (config.email.smtp) {
                document.getElementById('smtp-host').value = envValues.SMTP_HOST || config.email.smtp.host || '';
                document.getElementById('smtp-port').value = envValues.SMTP_PORT || config.email.smtp.port || '';
                document.getElementById('smtp-username').value = envValues.SMTP_USERNAME || config.email.smtp.username || '';
                
                if (envValues.SMTP_PASSWORD || config.email.smtp.configured) {
                    const passInput = document.getElementById('smtp-password');
                    passInput.placeholder = 'Password configured (leave blank to keep current)';
                    passInput.classList.add('configured');
                    
                    const userInput = document.getElementById('smtp-username');
                    userInput.classList.add('configured');
                }
            }
            
            if (envValues.SENDGRID_API_KEY || (config.email.sendgrid && config.email.sendgrid.configured)) {
                const keyInput = document.getElementById('sendgrid-key');
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
            }
            
            if (config.email.mailgun) {
                document.getElementById('mailgun-domain').value = envValues.MAILGUN_DOMAIN || config.email.mailgun.domain || '';
                if (envValues.MAILGUN_API_KEY || config.email.mailgun.configured) {
                    const keyInput = document.getElementById('mailgun-key');
                    keyInput.placeholder = 'API key configured (leave blank to keep current)';
                    keyInput.classList.add('configured');
                }
            }
        }
        
            
        if (envValues.OPENSTATES_API_KEY || (config.representatives && config.representatives.openstates_configured)) {
            const keyInput = document.getElementById('openstates-key');
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
            }
            
        
        if (config.scheduler) {
            document.getElementById('send-time').value = envValues.SCHEDULER_SEND_TIME || config.scheduler.SendTime || '09:00';
            document.getElementById('timezone').value = envValues.SCHEDULER_TIMEZONE || config.scheduler.Timezone || 'America/Los_Angeles';
            
            const schedulerEnabled = envValues.SCHEDULER_ENABLED;
            if (schedulerEnabled !== undefined) {
                document.getElementById('scheduler-enabled').checked = schedulerEnabled === 'true';
            } else {
                document.getElementById('scheduler-enabled').checked = config.scheduler.Enabled !== false;
            }
        }
        
        
        if (Object.keys(envValues).length > 0) {
            showStatus('Configuration loaded from .env file', 'success');
        }
        
    } catch (error) {
        showStatus('Failed to load configuration: ' + error.message, 'error');
    }
}


async function saveConfiguration() {
    const saveButton = document.getElementById('save-config');
    saveButton.disabled = true;
    saveButton.textContent = 'Saving...';
    
    const startTime = Date.now();
    const minDelay = 700; 
    
    let saveSuccess = false;
    let errorMessage = '';
    let configData = null;
    let openstatesKeyValue = '';
    
    try {
        
        const config = {
            user: {
                name: document.getElementById('user-name').value.trim(),
                email: document.getElementById('user-email').value.trim(),
                zip_code: document.getElementById('user-zip').value.trim(),
                send_copy_to_self: document.getElementById('send-copy-to-self').checked
            },
            ai: {
                provider: document.getElementById('ai-provider').value.trim()
            },
            email: {
                provider: document.getElementById('email-provider').value.trim()
            },
            representatives: {},
            scheduler: {
                send_time: document.getElementById('send-time').value.trim(),
                timezone: document.getElementById('timezone').value.trim(),
                enabled: document.getElementById('scheduler-enabled').checked
            },
            letter: {
                tone: document.getElementById('letter-tone').value.trim(),
                max_length: parseInt(document.getElementById('max-length').value),
                generation_method: document.getElementById('generation-method').value.trim()
            }
        };
        
        
        if (config.letter.generation_method === 'templates') {
            config.letter.template_config = {
                directory: document.getElementById('template-directory').value.trim(),
                rotation_strategy: document.getElementById('rotation-strategy').value.trim(),
                personalize: document.getElementById('personalize-templates').checked
            };
        }
        
        
        if (config.ai.provider === 'openai') {
            config.ai.openai = {
                model: document.getElementById('openai-model').value.trim()
            };
            
            const apiKey = document.getElementById('openai-key').value.trim();
            if (apiKey) {
                config.ai.openai.api_key = apiKey;
            }
        } else if (config.ai.provider === 'anthropic') {
            config.ai.anthropic = {
                model: document.getElementById('anthropic-model').value.trim()
            };
            
            const apiKey = document.getElementById('anthropic-key').value.trim();
            if (apiKey) {
                config.ai.anthropic.api_key = apiKey;
            }
        }
        
        if (config.email.provider === 'smtp') {
            config.email.smtp = {
                host: document.getElementById('smtp-host').value.trim(),
                port: parseInt(document.getElementById('smtp-port').value),
                username: document.getElementById('smtp-username').value.trim(),
                from: document.getElementById('smtp-username').value.trim()
            };
            
            const password = document.getElementById('smtp-password').value.trim();
            if (password) {
                config.email.smtp.password = password;
            }
        } else if (config.email.provider === 'sendgrid') {
            config.email.sendgrid = {
                from: document.getElementById('user-email').value.trim()
            };
            
            const apiKey = document.getElementById('sendgrid-key').value.trim();
            if (apiKey) {
                config.email.sendgrid.api_key = apiKey;
            }
        } else if (config.email.provider === 'mailgun') {
            config.email.mailgun = {
                domain: document.getElementById('mailgun-domain').value.trim(),
                from: `lettersmith@${document.getElementById('mailgun-domain').value.trim()}`
            };
            
            const apiKey = document.getElementById('mailgun-key').value.trim();
            if (apiKey) {
                config.email.mailgun.api_key = apiKey;
            }
        }
        
        
        const openstatesKey = document.getElementById('openstates-key').value.trim();
        if (openstatesKey) {
            config.representatives.openstates_api_key = openstatesKey;
        }
        
        
        configData = config;
        openstatesKeyValue = openstatesKey;
        
        
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            saveSuccess = true;
        } else {
            throw new Error(result.error || 'Failed to save configuration');
        }
    } catch (error) {
        errorMessage = error.message;
    } finally {
        
        const elapsed = Date.now() - startTime;
        const remainingDelay = Math.max(0, minDelay - elapsed);
        
        setTimeout(() => {
            
            saveButton.disabled = false;
            saveButton.textContent = 'Save Configuration';
            
            
            if (saveSuccess) {
                showStatus('Configuration saved successfully!', 'success');
                
                
                requestAnimationFrame(() => {
                    if (configData.ai.provider === 'openai' && document.getElementById('openai-key').value) {
                const keyInput = document.getElementById('openai-key');
                keyInput.value = '';
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
                    } else if (configData.ai.provider === 'anthropic' && document.getElementById('anthropic-key').value) {
                const keyInput = document.getElementById('anthropic-key');
                keyInput.value = '';
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
            }
            
                    if (configData.email.provider === 'smtp' && document.getElementById('smtp-password').value) {
                const passInput = document.getElementById('smtp-password');
                passInput.value = '';
                passInput.placeholder = 'Password configured (leave blank to keep current)';
                passInput.classList.add('configured');
                    } else if (configData.email.provider === 'sendgrid' && document.getElementById('sendgrid-key').value) {
                const keyInput = document.getElementById('sendgrid-key');
                keyInput.value = '';
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
                    } else if (configData.email.provider === 'mailgun' && document.getElementById('mailgun-key').value) {
                const keyInput = document.getElementById('mailgun-key');
                keyInput.value = '';
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
            }
            
                    if (openstatesKeyValue) {
                const keyInput = document.getElementById('openstates-key');
                keyInput.value = '';
                keyInput.placeholder = 'API key configured (leave blank to keep current)';
                keyInput.classList.add('configured');
            }
                });
            } else if (errorMessage) {
                showStatus('Error saving configuration: ' + errorMessage, 'error');
            }
        }, remainingDelay);
    }
}


async function testConfiguration() {
    const testButton = document.getElementById('test-config');
    testButton.disabled = true;
    testButton.textContent = 'Testing...';
    
    const startTime = Date.now();
    const minDelay = 700; 
    
    let testSuccess = false;
    let errorMessage = '';
    
    try {
        
        const errors = validateForm();
        if (errors.length > 0) {
            throw new Error('Please fix the following errors:\n' + errors.join('\n'));
        }
        
        // Build the same configuration object as saveConfiguration
        const config = {
            user: {
                name: document.getElementById('user-name').value.trim(),
                email: document.getElementById('user-email').value.trim(),
                zip_code: document.getElementById('user-zip').value.trim(),
                send_copy_to_self: document.getElementById('send-copy-to-self').checked
            },
            email: {
                provider: document.getElementById('email-provider').value.trim()
            }
        };
        
        // Add SMTP configuration if using SMTP
        if (config.email.provider === 'smtp') {
            config.email.smtp = {
                host: document.getElementById('smtp-host').value.trim(),
                port: parseInt(document.getElementById('smtp-port').value),
                username: document.getElementById('smtp-username').value.trim(),
                from: document.getElementById('smtp-username').value.trim()
            };
            
            const password = document.getElementById('smtp-password').value.trim();
            if (password) {
                config.email.smtp.password = password;
            }
            // Note: If password is empty but configured, backend will use stored password
        } else {
            // For now, only SMTP testing is implemented
            throw new Error('Email testing is currently only supported for SMTP. Please save configuration to validate other providers.');
        }
        
        
        const response = await fetch('/api/config/test-email', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            testSuccess = true;
        } else {
            throw new Error(result.error || 'Email test failed');
        }
    } catch (error) {
        errorMessage = error.message;
    } finally {
        
        const elapsed = Date.now() - startTime;
        const remainingDelay = Math.max(0, minDelay - elapsed);
        
        setTimeout(() => {
            
        testButton.disabled = false;
        testButton.textContent = 'Test Configuration';
            
            
            if (testSuccess) {
                showStatus('✅ Email test successful! Check your inbox for the test email.', 'success');
            } else if (errorMessage) {
                showStatus('❌ ' + errorMessage, 'error');
            }
        }, remainingDelay);
    }
}


function validateForm() {
    const errors = [];
    
    
    if (!document.getElementById('user-name').value) {
        errors.push('User name is required');
    }
    if (!document.getElementById('user-email').value) {
        errors.push('User email is required');
    }
    if (!document.getElementById('user-zip').value) {
        errors.push('ZIP code is required');
    }
    
    const generationMethod = document.getElementById('generation-method').value;
    
    
    if (generationMethod === 'ai') {
        const aiProvider = document.getElementById('ai-provider').value;
        if (!aiProvider) {
            errors.push('AI provider is required when using AI generation');
        } else if (aiProvider === 'openai') {
            const keyInput = document.getElementById('openai-key');
            
            if (!keyInput.value && !keyInput.classList.contains('configured')) {
                errors.push('OpenAI API key is required');
            }
        } else if (aiProvider === 'anthropic') {
            const keyInput = document.getElementById('anthropic-key');
            
            if (!keyInput.value && !keyInput.classList.contains('configured')) {
                errors.push('Anthropic API key is required');
            }
        }
    } else if (generationMethod === 'templates') {
        
        if (!document.getElementById('template-directory').value) {
            errors.push('Template directory is required when using templates');
        }
    }
    
    const emailProvider = document.getElementById('email-provider').value;
    if (!emailProvider) {
        errors.push('Email provider is required');
    } else if (emailProvider === 'smtp') {
        if (!document.getElementById('smtp-host').value) errors.push('SMTP host is required');
        if (!document.getElementById('smtp-port').value) errors.push('SMTP port is required');
        if (!document.getElementById('smtp-username').value) errors.push('SMTP username is required');
        
        const passInput = document.getElementById('smtp-password');
        
        if (!passInput.value && !passInput.classList.contains('configured')) {
            errors.push('SMTP password is required');
        }
    } else if (emailProvider === 'sendgrid') {
        const keyInput = document.getElementById('sendgrid-key');
        
        if (!keyInput.value && !keyInput.classList.contains('configured')) {
            errors.push('SendGrid API key is required');
        }
    } else if (emailProvider === 'mailgun') {
        const keyInput = document.getElementById('mailgun-key');
        
        if (!keyInput.value && !keyInput.classList.contains('configured')) {
            errors.push('Mailgun API key is required');
        }
        if (!document.getElementById('mailgun-domain').value) errors.push('Mailgun domain is required');
    }
    
    return errors;
}


function showStatus(message, type) {
    
    const existingNotifications = document.querySelectorAll('.floating-notification');
    existingNotifications.forEach(notification => {
        
        if (notification.dataset.timeoutId) {
            clearTimeout(parseInt(notification.dataset.timeoutId));
        }
        
        notification.classList.add('slide-up-quick');
        setTimeout(() => {
            notification.remove();
        }, 200);
    });
    
    
    setTimeout(() => {
        
        const notification = document.createElement('div');
        notification.className = `floating-notification ${type}`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        
        const timeoutId = setTimeout(() => {
            notification.classList.add('slide-up');
            setTimeout(() => {
                notification.remove();
            }, 300);
        }, 5000);
        
        
        notification.dataset.timeoutId = timeoutId;
    }, existingNotifications.length > 0 ? 250 : 0);
} 