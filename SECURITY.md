# Security Policy / 安全策略

## Report a vulnerability / 报告漏洞

Please do not publish exploitable security details in a public issue. Use GitHub's
private vulnerability reporting feature under **Security → Advisories → Report a
vulnerability**. Include the affected version, reproduction steps, and impact.

请勿在公开 Issue 中披露可利用的安全细节。请通过 GitHub 仓库的
**Security → Advisories → Report a vulnerability** 私下报告，并注明受影响版本、
复现步骤和影响。

## Scope / 范围

BiliGreen listens only on the loopback interface. Login cookies are kept in process
memory and are not intentionally written to disk. Reports about unintended remote
access, credential leakage, unsafe file access, or download integrity are in scope.

BiliGreen 仅监听本机回环地址，登录 Cookie 只保存在进程内存中。非预期远程访问、
凭据泄漏、不安全文件访问和下载完整性问题均属于安全报告范围。
