# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in dbdump, please report it by emailing the maintainer directly rather than opening a public issue.

**Contact:** [helgesverre@gmail.com](mailto:helgesverre@gmail.com)

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We aim to respond to security reports within 48 hours.

---

## Security Best Practices

### Password Handling

dbdump supports multiple methods for providing database credentials. Use the most secure method for your environment:

#### 1. Environment Variable (Recommended)

```bash
# Preferred: dbdump-specific variable
export DBDUMP_MYSQL_PWD=yourpassword
dbdump dump -u root -d mydb

# Alternative: standard MySQL variable (fallback)
export MYSQL_PWD=yourpassword
dbdump dump -u root -d mydb
```

**Pros:**
- Not visible in process lists
- Can be set in secure deployment systems
- `DBDUMP_MYSQL_PWD` avoids polluting standard MySQL environment
- Falls back to `MYSQL_PWD` for compatibility

**Cons:**
- Still in environment, accessible to same-user processes
- Must unset after use for maximum security

#### 2. MySQL Config File (Most Secure)

Create `~/.my.cnf` with restrictive permissions:

```bash
cat > ~/.my.cnf << 'EOF'
[client]
user=youruser
password=yourpassword
host=localhost
EOF

chmod 600 ~/.my.cnf
```

Then run dbdump without password flags:

```bash
dbdump dump -d mydb
```

**Pros:**
- Not in process list or environment
- Persistent across sessions
- File permissions protect the password

**Cons:**
- Password stored in plaintext file (but protected by permissions)
- Not suitable for multiple database credentials

#### 3. Command-Line Flag (Least Secure)

```bash
dbdump dump -u root -p yourpassword -d mydb
```

**WARNING:** Passwords provided as command-line flags may appear in:
- Shell history
- Process lists (visible to other users)
- Log files

Only use this method for local development or testing.

---

### Dump File Security

dbdump creates dump files with restrictive permissions (`0600`) by default, meaning:
- Only the file owner can read or write
- Other users cannot access the dump

**Important:** Dump files contain your database contents, including:
- User credentials
- Personal information
- Business data
- API keys stored in database

**Best Practices:**
- Encrypt dumps before transferring: `gpg -c dump.sql`
- Delete dumps after use: `shred -u dump.sql`
- Store dumps in encrypted volumes
- Use secure transfer methods (SCP, SFTP, not FTP)
- Set appropriate backup retention policies

---

### Network Security

When connecting to remote databases:

1. **Use SSH Tunnels** for untrusted networks:
```bash
# Create SSH tunnel
ssh -L 3307:localhost:3306 user@db-server

# Connect via tunnel
dbdump dump -H 127.0.0.1 -P 3307 -u dbuser -d mydb
```

2. **Use TLS/SSL Connections** when available
   - Note: dbdump currently doesn't support TLS flags directly
   - Use MySQL config file with SSL parameters

3. **Restrict Database Permissions**:
```sql
-- Create read-only user for dumps
CREATE USER 'dumper'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW VIEW, TRIGGER, LOCK TABLES ON mydb.* TO 'dumper'@'%';
FLUSH PRIVILEGES;
```

---

### Production Considerations

1. **Use Read Replicas** for production dumps to avoid impacting primary database
2. **Schedule dumps during low-traffic periods**
3. **Monitor resource usage** (CPU, disk I/O, network)
4. **Set up alerts** for failed dumps
5. **Test restore procedures** regularly

---

## Security Features in dbdump

### Implemented

- ✅ **Secure password passing** via `MYSQL_PWD` environment variable (not command-line args)
- ✅ **Restrictive file permissions** (0600) for dump files
- ✅ **Safe DSN construction** using mysql.Config with proper escaping
- ✅ **Connection timeouts** to prevent hanging on unreachable databases
- ✅ **Signal handling** for clean shutdown on Ctrl+C
- ✅ **No logging of credentials** in output

### Planned

- 🔄 Native TLS/SSL connection support
- 🔄 Integration with system keychains
- 🔄 Vault credential provider
- 🔄 Built-in encryption for dump files
- 🔄 Audit logging

---

## Known Limitations

1. **Password in memory:** Like all database tools, passwords exist in memory during execution
2. **Cleartext config files:** Config files store credentials in plaintext (use file permissions)
3. **No MFA support:** Database connections don't support multi-factor authentication
4. **Process memory dumps:** Password may be accessible via memory dumps (OS-level concern)

---

## Compliance

### GDPR Considerations

If your database contains personal data subject to GDPR:
- Ensure dump files are encrypted
- Implement appropriate retention policies
- Document data handling procedures
- Restrict access to authorized personnel
- Consider anonymizing personal data in development dumps

### PCI DSS

If dumping databases with payment card information:
- Never dump production credit card data to development
- Use data masking/anonymization (planned feature)
- Ensure compliance with PCI DSS requirements
- Audit all dump activities

---

## Version History

### v1.0.0 (Current - 2024-10-28)
- **[SECURITY]** Fixed password exposure in process lists (now uses MYSQL_PWD env var)
- **[SECURITY]** Dump files created with 0600 permissions
- **[SECURITY]** Safe DSN construction with proper escaping
- **[SECURITY]** Connection timeouts prevent hanging
- **[SECURITY]** Added DBDUMP_MYSQL_PWD as preferred environment variable

### v0.9.0 (2024-10-21)
- Initial public release with basic security measures

---

## Acknowledgments

Security improvements based on:
- OWASP Database Security Cheat Sheet
- MySQL Security Best Practices
- Industry-standard tools: Vitess, go-mysql
- Community feedback and security audits

---

**Last Updated:** 2024-10-28
