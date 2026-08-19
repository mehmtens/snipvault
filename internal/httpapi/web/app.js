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
const accountDialog = document.querySelector('#account-dialog');
let savedUser = JSON.parse(sessionStorage.getItem('snipvault_user') || 'null');
if (savedUser && !sessionStorage.getItem('snipvault_csrf')) {
	sessionStorage.removeItem('snipvault_user');
	sessionStorage.removeItem('snipvault_token');
	savedUser = null;
}
function syncAccount(user) { savedUser = user || null; accountButton.textContent = user?.username || 'Sign in'; myPastesButton.hidden = !user?.username; }
if (savedUser) syncAccount(savedUser);

accountButton.addEventListener('click', async () => {
	if (sessionStorage.getItem('snipvault_user')) {
		await loadProfile(); accountDialog.showModal();
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
  ['verify-panel','forgot-panel','reset-panel'].forEach(id => document.querySelector(`#${id}`).hidden = true);
}));

function showAuthPanel(id) { ['login-panel','register-panel','verify-panel','forgot-panel','reset-panel'].forEach(name => document.querySelector(`#${name}`).hidden = name !== id); document.querySelector('.auth-tabs').hidden = !['login-panel','register-panel'].includes(id); }
function storeSession(result) {
  if (!result?.user?.username || !result?.csrf_token) throw new Error('Your session could not be started. Refresh the page and try again.');
  sessionStorage.setItem('snipvault_csrf', result.csrf_token);
  sessionStorage.setItem('snipvault_user', JSON.stringify(result.user));
  syncAccount(result.user);
  authDialog.close();
}

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
    if (!response.ok) { if (endpoint === '/login' && response.status === 403) { document.querySelector('#verify-panel [name=email]').value = body.email; document.querySelector('#verify-email-label').textContent = body.email; showAuthPanel('verify-panel'); } throw new Error(result.error || 'Request failed'); }
    if (endpoint === '/register') { document.querySelector('#verify-panel [name=email]').value = result.email; document.querySelector('#verify-email-label').textContent = result.email; showAuthPanel('verify-panel'); document.querySelector('#verify-panel .code-input').focus(); return; }
		storeSession(result);
    form.reset();
  } catch (error) {
    status.textContent = error.message;
    status.classList.add('error');
  } finally { button.disabled = false; }
}
document.querySelector('#login-panel').addEventListener('submit', event => { event.preventDefault(); submitAuth(event.currentTarget, '/login'); });
document.querySelector('#register-panel').addEventListener('submit', event => { event.preventDefault(); submitAuth(event.currentTarget, '/register'); });
document.querySelector('#verify-panel').addEventListener('submit', async event => { event.preventDefault(); const form = event.currentTarget; const status = form.querySelector('.auth-message'); const response = await fetch('/verify-email',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(Object.fromEntries(new FormData(form)))}); const result=await response.json(); if(response.ok){storeSession(result)}else{status.textContent=result.error;status.classList.add('error')} });
document.querySelector('#resend-code').addEventListener('click', async () => { const email=document.querySelector('#verify-panel [name=email]').value; const status=document.querySelector('#verify-panel .auth-message'); const response=await fetch('/resend-verification',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})}); const result=await response.json(); status.textContent=result.message||result.error; status.classList.toggle('error',!response.ok); });
document.querySelector('#forgot-link').addEventListener('click',()=>showAuthPanel('forgot-panel')); document.querySelectorAll('.back-login').forEach(button=>button.addEventListener('click',()=>showAuthPanel('login-panel')));
document.querySelector('#forgot-panel').addEventListener('submit', async event=>{event.preventDefault();const form=event.currentTarget,email=new FormData(form).get('email'),status=form.querySelector('.auth-message');const response=await fetch('/forgot-password',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});const result=await response.json();if(!response.ok){status.textContent=result.error;status.classList.add('error');return}document.querySelector('#reset-panel [name=email]').value=email;showAuthPanel('reset-panel');});
document.querySelector('#reset-panel').addEventListener('submit',async event=>{event.preventDefault();const form=event.currentTarget,status=form.querySelector('.auth-message');const response=await fetch('/reset-password',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(Object.fromEntries(new FormData(form)))});if(response.ok){showAuthPanel('login-panel');document.querySelector('#login-panel .auth-message').textContent='Password updated. Sign in with your new password.';form.reset();return}const result=await response.json();status.textContent=result.error;status.classList.add('error');});

async function loadProfile(){const response=await fetch('/me/profile',{headers:authHeaders()});if(!response.ok)return;const user=await response.json();document.querySelector('#profile-form [name=username]').value=user.username;document.querySelector('#profile-form [name=email]').value=user.email;}
document.querySelector('#close-account').addEventListener('click',()=>accountDialog.close()); accountDialog.addEventListener('click',event=>{if(event.target===accountDialog)accountDialog.close()});
async function logout(){await fetch('/logout',{method:'POST',headers:authHeaders()});sessionStorage.removeItem('snipvault_csrf');sessionStorage.removeItem('snipvault_user');syncAccount(null);accountDialog.close();}
document.querySelector('#sign-out-button').addEventListener('click',logout);
document.querySelector('#profile-form').addEventListener('submit',async event=>{event.preventDefault();const form=event.currentTarget,status=form.querySelector('.auth-message'),username=new FormData(form).get('username');const response=await fetch('/me/profile',{method:'PATCH',headers:{'Content-Type':'application/json',...authHeaders()},body:JSON.stringify({username})});const result=await response.json();status.textContent=response.ok?'Profile saved.':result.error;status.classList.toggle('error',!response.ok);if(response.ok){sessionStorage.setItem('snipvault_user',JSON.stringify(result));syncAccount(result)}});
document.querySelector('#password-form').addEventListener('submit',async event=>{event.preventDefault();const form=event.currentTarget,status=form.querySelector('.auth-message');const response=await fetch('/me/password',{method:'POST',headers:{'Content-Type':'application/json',...authHeaders()},body:JSON.stringify(Object.fromEntries(new FormData(form)))});status.textContent=response.ok?'Password changed.':(await response.json()).error;status.classList.toggle('error',!response.ok);if(response.ok)form.reset()});
document.querySelector('#delete-account-form').addEventListener('submit',async event=>{event.preventDefault();if(!confirm('Delete your account and every paste permanently? This cannot be undone.'))return;const form=event.currentTarget,status=form.querySelector('.auth-message');const response=await fetch('/me/account',{method:'DELETE',headers:{'Content-Type':'application/json',...authHeaders()},body:JSON.stringify(Object.fromEntries(new FormData(form)))});if(response.ok){sessionStorage.clear();window.location.assign('/');return}status.textContent=(await response.json()).error;status.classList.add('error')});

const pastesDialog = document.querySelector('#pastes-dialog');
const pastesList = document.querySelector('#pastes-list');
const pasteFilters = document.querySelector('#paste-filters');
const pasteResultsStatus = document.querySelector('#paste-results-status');
const clearFilters = document.querySelector('#clear-filters');
document.querySelector('#close-pastes').addEventListener('click', () => pastesDialog.close());
pastesDialog.addEventListener('click', event => { if (event.target === pastesDialog) pastesDialog.close(); });

async function loadMyPastes() {
  pastesList.innerHTML = '<p class="empty-state">Loading your pastes…</p>';
  if (!pastesDialog.open) pastesDialog.showModal();
  try {
    const filters = new FormData(pasteFilters);
    const query = new URLSearchParams();
    ['q','language','visibility'].forEach(name => { const value=filters.get(name)?.trim(); if(value)query.set(name,value); });
    if (filters.get('favorite')) query.set('favorite','true');
    clearFilters.hidden = query.size === 0;
    const response = await fetch(`/me/pastes?${query}`, {headers: authHeaders()});
    const values = await response.json();
    if (!response.ok) throw new Error(values.error || 'Pastes could not be loaded');
    pasteResultsStatus.textContent = `${values.length} ${values.length === 1 ? 'paste' : 'pastes'} found`;
    if (!values.length) {
      const filtered = query.size > 0;
      pastesList.innerHTML = `<div class="empty-state"><strong>${filtered ? 'No matching pastes' : 'Your vault is ready'}</strong><span>${filtered ? 'Try a different search or clear the filters.' : 'Create your first paste and it will appear here.'}</span></div>`;
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
      const favorite = document.createElement('button');
      favorite.type = 'button'; favorite.className = 'favorite-button'; favorite.setAttribute('aria-pressed', String(value.is_favorite)); favorite.setAttribute('aria-label', value.is_favorite ? 'Remove from favorites' : 'Add to favorites');
      favorite.innerHTML = '<svg aria-hidden="true" viewBox="0 0 24 24"><path d="m12 3 2.78 5.63 6.22.9-4.5 4.39 1.06 6.2L12 17.2l-5.56 2.92 1.06-6.2L3 9.53l6.22-.9L12 3Z"/></svg>';
      favorite.addEventListener('click', async () => { favorite.disabled=true; const next=!value.is_favorite; const result=await fetch(`/pastes/${encodeURIComponent(value.slug)}/favorite`,{method:'PATCH',headers:{'Content-Type':'application/json',...authHeaders()},body:JSON.stringify({favorite:next})}); if(result.ok){value.is_favorite=next; await loadMyPastes()}else{favorite.disabled=false;window.alert('Favorite could not be updated.')} });
      actions.append(favorite, edit, remove);
      row.append(info, actions);
      pastesList.append(row);
    });
  } catch (error) {
    pastesList.innerHTML = `<p class="empty-state error">${error.message}</p>`;
  }
}

myPastesButton.addEventListener('click', loadMyPastes);
let filterTimer;
pasteFilters.addEventListener('input', () => { clearTimeout(filterTimer); filterTimer=setTimeout(loadMyPastes,250); });
pasteFilters.addEventListener('change', loadMyPastes);
pasteFilters.addEventListener('reset', () => setTimeout(loadMyPastes));
