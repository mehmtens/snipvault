const editorView = document.querySelector('#editor-view');
const pasteView = document.querySelector('#paste-view');
const content = document.querySelector('#content');
const lineNumbers = document.querySelector('#line-numbers');
const form = document.querySelector('#paste-form');
const message = document.querySelector('#form-message');
const createButton = document.querySelector('#create-button');
const authHeaders = () => {
	const csrfToken = sessionStorage.getItem('snipvault_csrf');
	return csrfToken ? {'X-CSRF-Token': csrfToken} : {};
};

const keywordSets = {
  go: 'break default func interface select case defer go map struct chan else goto package switch const fallthrough if range type continue for import return var',
  java: 'abstract assert boolean break byte case catch char class const continue default do double else enum extends final finally float for if implements import instanceof int interface long native new package private protected public return short static super switch synchronized this throw throws try void volatile while',
  c: 'auto break case char const continue default do double else enum extern float for goto if int long register return short signed sizeof static struct switch typedef union unsigned void volatile while',
  cpp: 'alignas alignof asm auto bool break case catch char class const constexpr continue default delete do double else enum explicit export extern false float for friend if inline int long namespace new noexcept nullptr operator private protected public return short signed sizeof static struct switch template this throw true try typedef typename union unsigned using virtual void volatile while',
  csharp: 'abstract as async await base bool break byte case catch char checked class const continue decimal default delegate do double else enum event explicit extern false finally fixed float for foreach if implicit in int interface internal is lock long namespace new null object operator out override params private protected public readonly ref return sealed short static string struct switch this throw true try typeof uint ulong unsafe using virtual void while',
  javascript: 'async await break case catch class const continue debugger default delete do else export extends false finally for from function get if import in instanceof let new null of return set static super switch this throw true try typeof undefined var void while yield',
  typescript: 'abstract any as async await boolean break case catch class const constructor continue declare default delete do else enum export extends false finally for from function get if implements import in infer instanceof interface keyof let namespace never new null number object of private protected public readonly return set static string super switch symbol this throw true try type typeof undefined unknown var void while yield',
  python: 'and as assert async await break class continue def del elif else except False finally for from global if import in is lambda None nonlocal not or pass raise return True try while with yield',
  sql: 'add all alter and any as asc between by case check column constraint create database default delete desc distinct drop exists foreign from full group having in index inner insert into is join key left like limit not null on or order outer primary right select set table truncate union unique update values view where',
  json: 'true false null'
};

function escapeHTML(value) {
  return value.replace(/[&<>]/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[char]));
}

function highlightCode(source, language) {
  const keywords = new Set((keywordSets[language] || '').split(' ').filter(Boolean));
  const tokenPattern = /("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`|\/\/[^\n]*|\/\*[\s\S]*?\*\/|#[^\n]*|\b\d+(?:\.\d+)?\b)/g;
  const plain = text => escapeHTML(text).replace(/\b[A-Za-z_$][\w$]*\b/g, word => keywords.has(word) ? `<span class="tok-keyword">${word}</span>` : word);
  let result = '';
  let cursor = 0;
  for (const match of source.matchAll(tokenPattern)) {
    result += plain(source.slice(cursor, match.index));
    const token = match[0];
    let type = 'number';
    if (token.startsWith('//') || token.startsWith('/*') || token.startsWith('#')) type = token.startsWith('#') && ['c','cpp'].includes(language) ? 'meta' : 'comment';
    else if (/^["'`]/.test(token)) type = 'string';
    result += `<span class="tok-${type}">${escapeHTML(token)}</span>`;
    cursor = match.index + token.length;
  }
  return result + plain(source.slice(cursor));
}

const numbersFor = (text) => Array.from({length: Math.max(1, text.split('\n').length)}, (_, i) => i + 1).join('\n');
content?.addEventListener('input', () => lineNumbers.textContent = numbersFor(content.value));
content?.addEventListener('keydown', (event) => {
  if (event.key !== 'Tab') return;
  event.preventDefault();
  content.setRangeText('  ', content.selectionStart, content.selectionEnd, 'end');
  lineNumbers.textContent = numbersFor(content.value);
});

form?.addEventListener('submit', async (event) => {
  event.preventDefault();
  message.classList.remove('error');
  message.textContent = 'Creating your link…';
  createButton.disabled = true;
  try {
    const editingSlug = sessionStorage.getItem('snipvault_edit_slug');
    const response = await fetch(editingSlug ? `/pastes/${encodeURIComponent(editingSlug)}` : '/pastes', {method: editingSlug ? 'PUT' : 'POST', headers: {'Content-Type': 'application/json', ...authHeaders()}, body: JSON.stringify({
      title: document.querySelector('#title').value,
      content: content.value,
      language: document.querySelector('#language').value,
      visibility: document.querySelector('#visibility').value,
      expires_in: document.querySelector('#expires-in').value
    })});
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || 'Paste could not be created');
    sessionStorage.removeItem('snipvault_edit_slug');
    sessionStorage.removeItem('snipvault_edit_draft');
    window.location.assign(`/p/${result.slug}`);
  } catch (error) {
    message.classList.add('error');
    message.textContent = error.message;
    createButton.disabled = false;
  }
});

async function loadPaste(slug) {
  editorView.hidden = true;
  pasteView.hidden = false;
  try {
    const response = await fetch(`/pastes/${encodeURIComponent(slug)}`, {headers: authHeaders()});
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || 'Paste could not be loaded');
    document.title = `${result.title || 'Untitled snippet'} — SnipVault`;
    document.querySelector('#paste-title').textContent = result.title || 'Untitled snippet';
    document.querySelector('#paste-language').textContent = result.language || 'text';
    document.querySelector('#paste-date').textContent = new Date(result.created_at).toLocaleString();
    document.querySelector('#paste-expiration').textContent = result.expires_at ? `Expires ${new Date(result.expires_at).toLocaleString()}` : 'Never expires';
    document.querySelector('#paste-visibility').textContent = result.visibility;
    document.querySelector('#paste-content').innerHTML = highlightCode(result.content, result.language);
    document.querySelector('#result-lines').textContent = numbersFor(result.content);
  } catch (error) {
    document.querySelector('#paste-title').textContent = 'Paste not found';
    document.querySelector('#paste-content').textContent = error.message;
  }
}

document.querySelector('#copy-button')?.addEventListener('click', async (event) => {
  await navigator.clipboard.writeText(window.location.href);
  event.currentTarget.textContent = 'Link copied';
  setTimeout(() => event.currentTarget.textContent = 'Copy link', 1800);
});

document.querySelector('#copy-code-button')?.addEventListener('click', async (event) => {
  await navigator.clipboard.writeText(document.querySelector('#paste-content').textContent);
  event.currentTarget.textContent = 'Code copied';
  setTimeout(() => event.currentTarget.textContent = 'Copy code', 1800);
});

async function rawBlob() {
  const slug = window.location.pathname.split('/').pop();
  const response = await fetch(`/raw/${encodeURIComponent(slug)}`, {headers: authHeaders()});
  if (!response.ok) throw new Error('Raw paste could not be loaded');
  return response.blob();
}

document.querySelector('#raw-button')?.addEventListener('click', async () => {
  try {
    const url = URL.createObjectURL(await rawBlob());
    window.open(url, '_blank', 'noopener');
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  } catch (error) { window.alert(error.message); }
});

document.querySelector('#download-button')?.addEventListener('click', async () => {
  try {
    const url = URL.createObjectURL(await rawBlob());
    const link = document.createElement('a');
    link.href = url;
    link.download = `${window.location.pathname.split('/').pop()}.txt`;
    link.click();
    URL.revokeObjectURL(url);
  } catch (error) { window.alert(error.message); }
});

const pathMatch = window.location.pathname.match(/^\/p\/([^/]+)$/);
if (pathMatch) loadPaste(pathMatch[1]);

const editDraft = JSON.parse(sessionStorage.getItem('snipvault_edit_draft') || 'null');
if (!pathMatch && editDraft) {
  document.querySelector('#title').value = editDraft.title || '';
  content.value = editDraft.content;
  document.querySelector('#language').value = editDraft.language || 'text';
  document.querySelector('#visibility').value = editDraft.visibility;
  if (editDraft.expires_at) {
    const remaining = new Date(editDraft.expires_at).getTime() - Date.now();
    document.querySelector('#expires-in').value = remaining <= 10*60*1000 ? '10m' : remaining <= 60*60*1000 ? '1h' : remaining <= 24*60*60*1000 ? '24h' : '7d';
  }
  lineNumbers.textContent = numbersFor(content.value);
  document.querySelector('#page-title').innerHTML = 'Refine the thought.<br><em>Keep the link.</em>';
  message.textContent = 'Editing an existing paste · the link stays the same';
  createButton.textContent = 'Save changes ↗';
}

const authDialog = document.querySelector('#auth-dialog');
const accountButton = document.querySelector('#account-button');
const myPastesButton = document.querySelector('#my-pastes-button');
let savedUser = JSON.parse(sessionStorage.getItem('snipvault_user') || 'null');
if (savedUser && !sessionStorage.getItem('snipvault_csrf')) {
	sessionStorage.removeItem('snipvault_user');
	sessionStorage.removeItem('snipvault_token');
	savedUser = null;
}
if (savedUser) { accountButton.textContent = `${savedUser.username} · Sign out`; myPastesButton.hidden = false; }

accountButton.addEventListener('click', async () => {
	if (sessionStorage.getItem('snipvault_user')) {
		await fetch('/logout', {method: 'POST', headers: authHeaders()});
		sessionStorage.removeItem('snipvault_csrf');
		sessionStorage.removeItem('snipvault_user');
    accountButton.textContent = 'Sign in';
    myPastesButton.hidden = true;
    return;
  }
  authDialog.showModal();
});
document.querySelector('#close-auth').addEventListener('click', () => authDialog.close());
authDialog.addEventListener('click', (event) => { if (event.target === authDialog) authDialog.close(); });

document.querySelectorAll('.auth-tab').forEach(tab => tab.addEventListener('click', () => {
  document.querySelectorAll('.auth-tab').forEach(item => item.classList.toggle('active', item === tab));
  document.querySelector('#login-panel').hidden = tab.dataset.panel !== 'login-panel';
  document.querySelector('#register-panel').hidden = tab.dataset.panel !== 'register-panel';
}));

async function submitAuth(form, endpoint) {
  const formData = new FormData(form);
  const status = form.querySelector('.auth-message');
  const button = form.querySelector('button[type="submit"]');
  status.textContent = endpoint === '/register' ? 'Creating account…' : 'Signing in…';
  status.classList.remove('error');
  button.disabled = true;
  try {
    const body = Object.fromEntries(formData.entries());
    const response = await fetch(endpoint, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || 'Request failed');
		sessionStorage.setItem('snipvault_csrf', result.csrf_token);
    sessionStorage.setItem('snipvault_user', JSON.stringify(result.user));
    accountButton.textContent = `${result.user.username} · Sign out`;
    myPastesButton.hidden = false;
    authDialog.close();
    form.reset();
  } catch (error) {
    status.textContent = error.message;
    status.classList.add('error');
  } finally { button.disabled = false; }
}
document.querySelector('#login-panel').addEventListener('submit', event => { event.preventDefault(); submitAuth(event.currentTarget, '/login'); });
document.querySelector('#register-panel').addEventListener('submit', event => { event.preventDefault(); submitAuth(event.currentTarget, '/register'); });

const pastesDialog = document.querySelector('#pastes-dialog');
const pastesList = document.querySelector('#pastes-list');
document.querySelector('#close-pastes').addEventListener('click', () => pastesDialog.close());
pastesDialog.addEventListener('click', event => { if (event.target === pastesDialog) pastesDialog.close(); });

async function loadMyPastes() {
  pastesList.innerHTML = '<p class="empty-state">Loading your pastes…</p>';
  pastesDialog.showModal();
  try {
    const response = await fetch('/me/pastes', {headers: authHeaders()});
    const values = await response.json();
    if (!response.ok) throw new Error(values.error || 'Pastes could not be loaded');
    if (!values.length) {
      pastesList.innerHTML = '<p class="empty-state">No pastes yet. Create one and it will appear here.</p>';
      return;
    }
    pastesList.innerHTML = '';
    values.forEach(value => {
      const row = document.createElement('article');
      row.className = 'paste-row';
      const info = document.createElement('a');
      info.href = `/p/${value.slug}`;
      const title = document.createElement('strong');
      title.textContent = value.title || 'Untitled snippet';
      const meta = document.createElement('span');
      const expiry = value.expires_at ? `expires ${new Date(value.expires_at).toLocaleString()}` : 'never expires';
      meta.textContent = `${value.language || 'text'} · ${value.visibility} · ${expiry}`;
      info.append(title, meta);
      const remove = document.createElement('button');
      remove.type = 'button';
      remove.textContent = 'Delete';
      remove.className = 'delete-button';
      remove.addEventListener('click', async () => {
        if (!window.confirm(`Delete “${value.title || 'Untitled snippet'}”?`)) return;
        const result = await fetch(`/pastes/${encodeURIComponent(value.slug)}`, {method: 'DELETE', headers: authHeaders()});
        if (result.ok) row.remove(); else window.alert('Paste could not be deleted.');
      });
      const edit = document.createElement('button');
      edit.type = 'button';
      edit.textContent = 'Edit';
      edit.className = 'edit-button';
      edit.addEventListener('click', () => {
        sessionStorage.setItem('snipvault_edit_slug', value.slug);
        sessionStorage.setItem('snipvault_edit_draft', JSON.stringify(value));
        window.location.assign('/');
      });
      const actions = document.createElement('div');
      actions.className = 'row-actions';
      actions.append(edit, remove);
      row.append(info, actions);
      pastesList.append(row);
    });
  } catch (error) {
    pastesList.innerHTML = `<p class="empty-state error">${error.message}</p>`;
  }
}

myPastesButton.addEventListener('click', loadMyPastes);
