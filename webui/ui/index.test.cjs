const { test } = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const vm = require('node:vm');

function setup() {
  const elements = new Map();
  const element = id => {
    if (!elements.has(id)) elements.set(id, {
      value: '', innerHTML: '', textContent: '', style: {}, disabled: false,
      classList: { add() {}, remove() {}, toggle() {} },
      querySelectorAll: () => [],
    });
    return elements.get(id);
  };
  const listeners = {};
  const context = vm.createContext({
    document: {
      getElementById: element, querySelectorAll: () => [],
      documentElement: { setAttribute() {} },
    },
    localStorage: { getItem: () => 'dark', setItem() {} },
    window: { matchMedia: () => ({ matches: true }), addEventListener: (name, fn) => listeners[name] = fn },
    confirm: () => false, console, setInterval: () => 1,
    fetch: async () => { throw new Error('Unexpected request'); },
  });
  const html = readFileSync(__dirname + '/index.html', 'utf8');
  const script = html.match(/<script>([\s\S]*?)<\/script>/)[1];
  vm.runInContext(script.split('  // init')[0], context);
  return { context, element, listeners, run: code => vm.runInContext(code, context) };
}

test('empty history differs from a filter with no matches', () => {
  const { run, element } = setup();
  run('renderTable([])');
  assert.match(element('table-container').innerHTML, /No backup data yet/);
  run('allEntries = [{repo_name: "example"}]; renderTable([])');
  assert.match(element('table-container').innerHTML, /No entries match/);
});

test('navigation preserves edits and cancelled switching or reload retains content', async () => {
  const { run, element, listeners } = setup();
  run('configLoaded = true; savedConfigContent = "original"');
  element('config-content').value = 'edited';
  run('showPage("dashboard"); showPage("config"); selectConfigFile(1)');
  await run('loadConfig()');
  assert.equal(run('activeConfigFile'), 0);
  assert.equal(element('config-content').value, 'edited');
  let prevented = false;
  listeners.beforeunload({ preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
});

test('saving an older snapshot preserves newer unsaved edits', async () => {
  const { context, run, element } = setup();
  run('configLoaded = true; savedConfigContent = "original"');
  element('config-content').value = 'first edit';
  let finish;
  context.fetch = () => new Promise(resolve => finish = resolve);
  const saving = run('saveConfig()');
  element('config-content').value = 'newer edit';
  finish({ ok: true });
  await saving;
  assert.equal(run('configDirty()'), true);
  assert.equal(run('savedConfigContent'), 'first edit');
});

test('polling detects scheduled runs even when the UI was idle', async () => {
  const { context, run, element } = setup();
  context.fetch = async () => ({ ok: true, json: async () => ({ running: true }) });
  await run('pollRunning()');
  assert.equal(element('run-status').style.display, 'inline-flex');
});


test('setup opens the editor for an empty installation and clears after loading config', async () => {
  const { context, run, element } = setup();
  let configs = [];
  context.fetch = async path => ({
    ok: true,
    json: async () => path === '/api/configs' ? configs : [{ index: 0, name: 'conf.yml', path: '/config/conf.yml' }],
    text: async () => '',
  });
  await run('loadConfigs()');
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(element('setup-notice').style.display, '');
  assert.equal(run('setupShown'), true);
  assert.equal(run('configLoaded'), true);
  assert.equal(element('config-path').textContent, 'Configuration file: /config/conf.yml');
  configs = [{ index: 0, name: 'conf.yml', sources: 1, dests: 1 }];
  await run('loadConfigs()');
  assert.equal(element('setup-notice').style.display, 'none');
  assert.equal(run('hasConfigs'), true);
});
