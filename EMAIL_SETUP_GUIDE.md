# 📧 Lettersmith Email Setup Guide

This guide covers how to set up email sending with Lettersmith using different providers and deployment strategies.

## Quick Setup (Recommended for Most Users)

### 🚀 ProtonMail Bridge (Easiest for ProtonMail users)
1. **Install ProtonMail Bridge** on your host machine:
   - Download from [proton.me/mail/bridge](https://proton.me/mail/bridge)
   - Install and log in with your ProtonMail account
   
2. **Configure in Lettersmith**:
   - Choose "SMTP (ProtonMail, Gmail, etc.)" as email provider
   - Select "ProtonMail Bridge (Local)" preset
   - Enter your ProtonMail email as username
   - Use the Bridge password from the Bridge app (NOT your ProtonMail password)

3. **That's it!** Bridge runs on your host, Docker connects via `127.0.0.1:1025`

### 📨 Gmail/Outlook/Yahoo (Direct SMTP)
1. **Set up App Password** (required for 2FA-enabled accounts):
   - **Gmail**: Google Account → Security → App passwords → Generate for "Mail"
   - **Outlook**: If 2FA enabled, create app password in security settings
   - **Yahoo**: Account Settings → Security → Generate app password for "Desktop app"

2. **Configure in Lettersmith**:
   - Choose your provider preset (Gmail/Outlook/Yahoo)
   - Enter your email address as username
   - Use the generated app password (NOT your regular password)

## Advanced: All-in-One Docker Solution

For users who want **everything containerized** including ProtonMail Bridge:

### 🐳 Option A: Add Bridge to Your Docker Compose

Add this service to your `docker-compose.yml`:

```yaml
services:
  # ... your existing services ...
  
  protonmail-bridge:
    image: shenxn/protonmail-bridge:latest
    container_name: protonmail-bridge
    volumes:
      - protonmail_data:/root
    ports:
      - "127.0.0.1:1025:25"
      - "127.0.0.1:1143:143"
    restart: unless-stopped
    networks:
      - lettersmith_network

  lettersmith:
    # ... your existing lettersmith config ...
    depends_on:
      - protonmail-bridge
    environment:
      - SMTP_HOST=protonmail-bridge  # Use container name
      - SMTP_PORT=25
      # ... other env vars

volumes:
  protonmail_data:

networks:
  lettersmith_network:
```

### 🔧 Option B: Pre-built All-in-One Image

You can also use existing all-in-one images:

```yaml
services:
  protonmail-bridge:
    image: ghcr.io/videocurio/proton-mail-bridge:latest
    volumes:
      - protonmail_data:/root
    ports:
      - "127.0.0.1:12025:25"
      - "127.0.0.1:12143:143"
    restart: unless-stopped
```

### 🛠️ Setup Process for Docker Bridge

1. **First-time setup** (interactive):
   ```bash
   # Start the containers
   docker compose up -d
   
   # Initialize ProtonMail Bridge
   docker exec -it protonmail-bridge /bin/bash
   
   # Kill default bridge and start interactive mode
   pkill bridge
   /app/bridge --cli
   ```

2. **In the bridge CLI**:
   ```
   >>> login
   Username: your-email@proton.me
   Password: [your-protonmail-password]
   Two factor code: 123456
   >>> info  # Note the credentials
   >>> exit
   ```

3. **Update Lettersmith configuration**:
   - Use the container network hostname (`protonmail-bridge`)
   - Use port `25` (internal container port)
   - Use the credentials from `info` command

4. **Restart to apply**:
   ```bash
   docker compose restart
   ```

## Provider-Specific Instructions

### ProtonMail
- **Bridge Password**: Found in ProtonMail Bridge app, NOT your account password
- **Port**: 1025 (host bridge) or 25 (container bridge)
- **Security**: Self-signed certificates (handled automatically)

### Gmail
- **Requirements**: 2FA must be enabled for App Passwords
- **Username**: Full Gmail address
- **Password**: 16-character App Password from Google Account settings
- **Settings**: `smtp.gmail.com:587` with STARTTLS

### Outlook/Hotmail
- **Username**: Full Outlook/Hotmail address
- **Password**: Regular password or App Password if 2FA enabled
- **Settings**: `smtp-mail.outlook.com:587` with STARTTLS

### Yahoo
- **Requirements**: App Password required (Yahoo doesn't allow regular passwords)
- **Username**: Full Yahoo email address
- **Password**: App Password from Yahoo Account Security settings
- **Settings**: `smtp.mail.yahoo.com:587` with STARTTLS

## Security Considerations

### ✅ Secure Practices
- Use App Passwords instead of regular passwords when available
- Run ProtonMail Bridge on trusted networks only
- Use strong, unique passwords for email accounts
- Enable 2FA on all email accounts

### ⚠️ Docker Bridge Security Notes
- Container bridge exposes SMTP ports to Docker network
- Ensure proper Docker network isolation
- Consider VPN for remote access instead of exposing ports publicly
- Regularly update bridge container images

## Troubleshooting

### Common Issues

**"Authentication failed"**
- Verify you're using App Password, not regular password
- For ProtonMail: Ensure Bridge is running and you're using Bridge password
- Check username format (full email address)

**"Connection refused"**
- For Bridge: Verify Bridge is running (`docker ps` or check host process)
- Check port configuration (1025 for host bridge, 25 for container)
- Verify host/container connectivity

**"Certificate errors"**
- ProtonMail Bridge uses self-signed certificates (this is normal)
- Lettersmith handles this automatically for localhost connections

### Testing Commands

**Test SMTP connectivity**:
```bash
# From host (if bridge on host)
telnet 127.0.0.1 1025

# From container (if using container bridge)
docker exec lettersmith telnet protonmail-bridge 25
```

**Check Bridge status**:
```bash
# Host bridge
ps aux | grep bridge

# Container bridge
docker exec protonmail-bridge ps aux | grep bridge
```

## Performance & Reliability

### Email Sending Limits
- **Gmail**: 100 emails/day (free), 2000/day (Google Workspace)
- **ProtonMail**: Based on your plan (500/day for Plus)
- **Outlook**: 300 emails/day for personal accounts
- **Yahoo**: ~500 emails/day

### Recommendations
- Use SendGrid/Mailgun for high-volume sending
- ProtonMail Bridge is ideal for privacy-focused, moderate volume
- Gmail/Outlook work well for personal use and testing

## Migration & Backup

### Backing up Bridge Data
```bash
# Create backup of ProtonMail Bridge data
docker run --rm -v protonmail_data:/source -v $(pwd):/backup alpine tar czf /backup/protonmail-backup.tar.gz -C /source .

# Restore backup
docker run --rm -v protonmail_data:/target -v $(pwd):/backup alpine tar xzf /backup/protonmail-backup.tar.gz -C /target
```

### Switching Providers
1. Update email provider in Lettersmith UI
2. Test configuration before going live
3. Consider keeping backup provider configured
4. Update any monitoring/alerting for new provider limits

---

## Need Help?

- **ProtonMail Bridge Issues**: [Proton Support](https://proton.me/support)
- **Gmail App Passwords**: [Google Support](https://support.google.com/accounts/answer/185833)
- **Docker Issues**: Check container logs with `docker compose logs`
- **Lettersmith Issues**: Check application logs and test email configuration in UI 