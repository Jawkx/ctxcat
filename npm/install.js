const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// The destination for the binary
const BIN_DIR = __dirname;
const BIN_PATH = path.join(BIN_DIR, 'ctxcat-bin');

function getPlatformMapping() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    'darwin': 'Darwin',
    'linux': 'Linux',
    'win32': 'Windows'
  };

  const archMap = {
    'x64': 'x86_64',
    'arm64': 'arm64',
    'ia32': 'i386'
  };

  if (!platformMap[platform] || !archMap[arch]) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return { os: platformMap[platform], arch: archMap[arch] };
}

async function downloadAndExtract() {
  try {
    const pkg = require('./package.json');
    const version = pkg.version;
    const { os, arch } = getPlatformMapping();
    const isWindows = os === 'Windows';
    const extension = isWindows ? 'zip' : 'tar.gz';

    // Construct the file name and URL based on your GoReleaser config
    const fileName = `ctxcat_${os}_${arch}.${extension}`;
    const url = `https://github.com/Jawkx/ctxcat/releases/download/v${version}/${fileName}`;

    console.log(`Downloading ctxcat binary from: ${url}`);

    // Create a temporary file to save the download
    const tempFilePath = path.join(BIN_DIR, fileName);
    const fileStream = fs.createWriteStream(tempFilePath);

    const response = await new Promise((resolve, reject) => {
      https.get(url, (res) => {
        if (res.statusCode === 302) {
          https.get(res.headers.location, resolve).on('error', reject);
        } else {
          resolve(res);
        }
      }).on('error', reject);
    });


    if (response.statusCode !== 200) {
      throw new Error(`Download failed with status code: ${response.statusCode}`);
    }

    response.pipe(fileStream);

    await new Promise((resolve, reject) => {
      fileStream.on('finish', resolve);
      fileStream.on('error', reject);
    });

    console.log('Download complete. Extracting binary...');

    // Use system tools to extract. This avoids adding extra npm dependencies.
    if (isWindows) {
      execSync(`tar -xf "${tempFilePath}" -C "${BIN_DIR}"`);
    } else {
      execSync(`tar -xzf "${tempFilePath}" -C "${BIN_DIR}"`);
    }

    // GoReleaser might put the binary in a subdirectory. Let's find it.
    const executableName = isWindows ? 'ctxcat.exe' : 'ctxcat';
    let foundPath = path.join(BIN_DIR, executableName); // Check root first

    if (!fs.existsSync(foundPath)) {
      // Fallback to checking common subdirectories if GoReleaser changes behavior
      const possibleSubdirs = fs.readdirSync(BIN_DIR).filter(f => fs.statSync(path.join(BIN_DIR, f)).isDirectory());
      for (const subdir of possibleSubdirs) {
        const potentialPath = path.join(BIN_DIR, subdir, executableName);
        if (fs.existsSync(potentialPath)) {
          foundPath = potentialPath;
          break;
        }
      }
    }

    if (!fs.existsSync(foundPath)) {
      throw new Error(`Could not find executable '${executableName}' after extraction.`);
    }

    // Move binary to the final destination and make it executable
    fs.renameSync(foundPath, BIN_PATH);
    if (!isWindows) {
      fs.chmodSync(BIN_PATH, '755');
    }

    // Clean up
    fs.unlinkSync(tempFilePath);
    const extractedDirs = fs.readdirSync(BIN_DIR).filter(f => fs.statSync(path.join(BIN_DIR, f)).isDirectory() && f.startsWith('ctxcat_'));
    for (const dir of extractedDirs) {
      fs.rmdirSync(path.join(BIN_DIR, dir), { recursive: true });
    }

    console.log('ctxcat installed successfully!');

  } catch (error) {
    console.error('Failed to install ctxcat binary.', error);
    process.exit(1);
  }
}

downloadAndExtract();
