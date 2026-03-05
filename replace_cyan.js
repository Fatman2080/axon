const fs = require('fs');
const path = require('path');

const dirPath = path.join(__dirname, 'frontend-www/src');

function replaceInFile(filePath) {
  let content = fs.readFileSync(filePath, 'utf8');
  let original = content;

  // Replace cyan variables and classes
  content = content.replace(/var\(--cyan\)/g, 'var(--neon-green)');
  content = content.replace(/var\(--cyan-dim\)/g, 'var(--neon-green-dim)');
  content = content.replace(/tag-cyan/g, 'tag-neon');
  content = content.replace(/text-cyan/g, 'text-neon-green');
  content = content.replace(/bg-cyan/g, 'bg-neon-green');
  content = content.replace(/bg-cyan-dim/g, 'bg-neon-green-dim');
  content = content.replace(/#00F0FF/g, '#00FF41'); // replace cyan hex if any
  content = content.replace(/#00f0ff/gi, '#00FF41');
  
  if (content !== original) {
    fs.writeFileSync(filePath, content, 'utf8');
    console.log(`Updated $\\{filePath\\}`);
  }
}

function traverseDir(dir) {
  const files = fs.readdirSync(dir);
  for (const file of files) {
    const fullPath = path.join(dir, file);
    const stat = fs.statSync(fullPath);
    if (stat.isDirectory()) {
      traverseDir(fullPath);
    } else if (fullPath.endsWith('.tsx') || fullPath.endsWith('.ts') || fullPath.endsWith('.css')) {
      replaceInFile(fullPath);
    }
  }
}

traverseDir(dirPath);
