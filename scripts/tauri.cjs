const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '..');

function findGoBin() {
  const candidates = [
    process.env.GOROOT ? path.join(process.env.GOROOT, 'bin') : '',
    'D:\\DevTools\\go\\bin',
    'C:\\Program Files\\Go\\bin',
  ];

  const whereResult = spawnSync('where.exe', ['go.exe'], {
    encoding: 'utf8',
    shell: false,
  });
  if (whereResult.status === 0) {
    const goPath = whereResult.stdout.split(/\r?\n/u).find(Boolean);
    if (goPath) {
      candidates.unshift(path.dirname(goPath.trim()));
    }
  }

  return candidates.find((candidate) => (
    candidate && fs.existsSync(path.join(candidate, 'go.exe'))
  )) || '';
}

const goBinPath = findGoBin();

function withGoPath(options = {}) {
  const currentPath = process.env.PATH || '';
  const pathValue = fs.existsSync(goBinPath)
    ? `${goBinPath}${path.delimiter}${currentPath}`
    : currentPath;

  return {
    ...options,
    env: {
      ...process.env,
      ...options.env,
      PATH: pathValue,
    },
  };
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    stdio: 'inherit',
    shell: false,
    ...withGoPath(options),
  });

  if (result.error) {
    throw result.error;
  }

  if (result.status !== 0) {
    process.exit(typeof result.status === 'number' ? result.status : 1);
  }
}

function runFinal(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    stdio: 'inherit',
    shell: false,
    ...withGoPath(options),
  });

  if (result.error) {
    throw result.error;
  }

  process.exit(typeof result.status === 'number' ? result.status : 1);
}

function runTauriDirect() {
  run('npm.cmd', ['run', 'sync-version'], { shell: process.platform === 'win32' });
  runFinal('npx.cmd', ['tauri', ...process.argv.slice(2)], { shell: process.platform === 'win32' });
}

function findVcvars64() {
  if (process.env.VCVARS64_PATH && fs.existsSync(process.env.VCVARS64_PATH)) {
    return process.env.VCVARS64_PATH;
  }

  const vswherePath = path.join(
    process.env['ProgramFiles(x86)'] || 'C:\\Program Files (x86)',
    'Microsoft',
    'Visual Studio',
    'Installer',
    'vswhere.exe',
  );

  if (fs.existsSync(vswherePath)) {
    const result = spawnSync(vswherePath, [
      '-latest',
      '-products',
      '*',
      '-requires',
      'Microsoft.VisualStudio.Component.VC.Tools.x86.x64',
      '-property',
      'installationPath',
    ], {
      encoding: 'utf8',
      shell: false,
    });

    const installationPath = result.status === 0 ? result.stdout.trim() : '';
    const discoveredPath = installationPath
      ? path.join(installationPath, 'VC', 'Auxiliary', 'Build', 'vcvars64.bat')
      : '';

    if (discoveredPath && fs.existsSync(discoveredPath)) {
      return discoveredPath;
    }
  }

  const defaultPath = 'C:\\Program Files (x86)\\Microsoft Visual Studio\\2022\\BuildTools\\VC\\Auxiliary\\Build\\vcvars64.bat';
  return fs.existsSync(defaultPath) ? defaultPath : '';
}

if (process.platform !== 'win32') {
  run('npm', ['run', 'sync-version']);
  runFinal('npx', ['tauri', ...process.argv.slice(2)]);
}

const vcvars64Path = findVcvars64();

if (!vcvars64Path) {
  console.warn('vcvars64.bat not found, falling back to the existing shell environment.');
  runTauriDirect();
}

const tempScriptPath = path.join(os.tmpdir(), `aiswitch-tauri-${process.pid}.cmd`);
const tauriCliPath = path.join(repoRoot, 'node_modules', '.bin', 'tauri.cmd');
const tauriArgs = process.argv.slice(2);

if (!fs.existsSync(tauriCliPath)) {
  console.warn('Local tauri CLI not found, falling back to the existing shell environment.');
  runTauriDirect();
}

const quotedArgs = tauriArgs.map((arg) => {
  if (/[\s"]/u.test(arg)) {
    return `"${arg.replace(/"/g, '""')}"`;
  }
  return arg;
});
const scriptBody = [
  '@echo off',
  `set "PATH=${goBinPath};%PATH%"`,
  `call "${vcvars64Path}"`,
  'if errorlevel 1 exit /b %errorlevel%',
  'call npm.cmd run sync-version',
  'if errorlevel 1 exit /b %errorlevel%',
  `call "${tauriCliPath}" ${quotedArgs.join(' ')}`.trim(),
].join('\r\n');

fs.writeFileSync(tempScriptPath, scriptBody);

try {
  runFinal('cmd.exe', ['/d', '/c', tempScriptPath]);
} finally {
  fs.rmSync(tempScriptPath, { force: true });
}
