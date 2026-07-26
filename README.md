# Couple Mini Program

This workspace contains:

- `miniprogram/`: native WeChat Mini Program UI built from the provided screenshots.
- `backend/`: Gin + GORM + MySQL REST backend with seeded demo data, CRUD management, and task state transitions.

Open `miniprogram` in WeChat Developer Tools. Run the backend with:

```bash
cd backend
# Linux
./start.sh

# Windows PowerShell
.\start.ps1
```

To upload the mini program with CI tooling:

```bash
cd miniprogram
npm install
cp upload.config.example.json upload.config.json
# fill in privateKeyPath, then run
npm run upload
```
