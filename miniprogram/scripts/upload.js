const fs = require("fs");
const path = require("path");

function fail(message) {
  console.error(message);
  process.exit(1);
}

const rootDir = path.resolve(__dirname, "..");
const uploadConfigPath = path.join(rootDir, "upload.config.json");
const exampleConfigPath = path.join(rootDir, "upload.config.example.json");

if (!fs.existsSync(uploadConfigPath)) {
  fail(
    [
      "Missing upload.config.json.",
      `Copy ${path.basename(exampleConfigPath)} to upload.config.json and fill in the private key path first.`
    ].join(" ")
  );
}

const rawConfig = fs.readFileSync(uploadConfigPath, "utf8");
let uploadConfig;
try {
  uploadConfig = JSON.parse(rawConfig);
} catch (error) {
  fail(`Invalid upload.config.json: ${error.message}`);
}

const projectConfigPath = path.join(rootDir, "project.config.json");
if (!fs.existsSync(projectConfigPath)) {
  fail("Missing project.config.json.");
}

const projectConfig = JSON.parse(fs.readFileSync(projectConfigPath, "utf8"));
const appid = uploadConfig.appid || projectConfig.appid;
if (!appid) {
  fail("Missing appid in upload.config.json or project.config.json.");
}

const privateKeyPath = path.resolve(rootDir, uploadConfig.privateKeyPath || "");
if (!uploadConfig.privateKeyPath || !fs.existsSync(privateKeyPath)) {
  fail(`Missing private key file: ${privateKeyPath}`);
}

const ci = require("miniprogram-ci");
const projectPath = path.resolve(rootDir, uploadConfig.projectPath || ".");
const version = process.env.MINIPROGRAM_VERSION || uploadConfig.version || "1.0.0";
const desc = process.env.MINIPROGRAM_DESC || uploadConfig.desc || "ci upload";
const robot = Number(process.env.MINIPROGRAM_ROBOT || uploadConfig.robot || 1);

async function main() {
  const project = new ci.Project({
    appid,
    type: "miniProgram",
    projectPath,
    privateKeyPath,
    ignores: ["node_modules/**/*"]
  });

  await ci.upload({
    project,
    version,
    desc,
    robot,
    setting: {
      es6: true,
      minify: true,
      codeProtect: false,
      autoPrefixWXSS: true
    },
    onProgressUpdate(message) {
      if (message && typeof message === "object") {
        console.log(JSON.stringify(message));
        return;
      }
      console.log(message);
    }
  });

  console.log(`Upload completed for ${appid}, version ${version}.`);
}

main().catch((error) => {
  fail(`Upload failed: ${error.message}`);
});
