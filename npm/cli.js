#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

const binaryPath = path.join(__dirname, 'ctxcat-bin');

const child = spawn(
  binaryPath,
  process.argv.slice(2), // Pass all arguments to the binary
  { stdio: 'inherit' }   // Connect stdin, stdout, and stderr
);

child.on('exit', (code) => {
  process.exit(code);
});

child.on('error', (err) => {
  console.error('Failed to start the ctxcat binary:', err);
  process.exit(1);
});
