<script>
  import { onMount } from 'svelte';
  import {
    listWebhooks, createWebhook, deleteWebhook, activateWebhook, deactivateWebhook,
    getExportURL, exportBooks
  } from '$lib/api.js';
  import { notify } from '$lib/stores.js';

  let webhooks = [];
  let webhooksLoading = true;
  let showWebhookForm = false;
  let webhookForm = { url: '', secret: '', events: [], description: '' };
  let webhookFormError = '';
  let webhookSubmitting = false;

  const allEvents = ['book.created', 'book.updated', 'book.deleted', 'book.text_extracted'];

  let exportTag = '';
  let exportCategory = '';
  let exportLanguage = '';
  let exportLoading = false;

  onMount(async () => { await loadWebhooks(); });

  async function loadWebhooks() {
    webhooksLoading = true;
    try {
      const result = await listWebhooks();
      webhooks = result.data || [];
    } catch (err) {
      notify.error('Failed to load webhooks: ' + err.message);
    } finally {
      webhooksLoading = false;
    }
  }

  async function handleCreateWebhook() {
    webhookFormError = '';
    if (!webhookForm.url) { webhookFormError = 'URL is required'; return; }
    if (webhookForm.secret.length < 16) { webhookFormError = 'Secret must be at least 16 characters'; return; }
    if (webhookForm.events.length === 0) { webhookFormError = 'Select at least one event'; return; }

    webhookSubmitting = true;
    try {
      await createWebhook(webhookForm);
      notify.success('Webhook created');
      showWebhookForm = false;
      webhookForm = { url: '', secret: '', events: [], description: '' };
      await loadWebhooks();
    } catch (err) {
      webhookFormError = err.message;
    } finally {
      webhookSubmitting = false;
    }
  }

  async function handleDeleteWebhook(id, url) {
    if (!confirm(`Delete webhook: ${url}?`)) return;
    try {
      await deleteWebhook(id);
      notify.success('Webhook deleted');
      await loadWebhooks();
    } catch (err) {
      notify.error('Delete failed: ' + err.message);
    }
  }

  async function toggleWebhook(wh) {
    try {
      if (wh.active) {
        await deactivateWebhook(wh.id);
        notify.info('Webhook deactivated');
      } else {
        await activateWebhook(wh.id);
        notify.success('Webhook activated');
      }
      await loadWebhooks();
    } catch (err) {
      notify.error('Failed: ' + err.message);
    }
  }

  function toggleEvent(event) {
    if (webhookForm.events.includes(event)) {
      webhookForm.events = webhookForm.events.filter(e => e !== event);
    } else {
      webhookForm.events = [...webhookForm.events, event];
    }
  }

  function generateSecret() {
    const arr = new Uint8Array(24);
    crypto.getRandomValues(arr);
    webhookForm.secret = Array.from(arr, b => b.toString(16).padStart(2, '0')).join('');
  }

  async function handleExport() {
    exportLoading = true;
    try {
      const params = {};
      if (exportTag) params.tag = exportTag;
      if (exportCategory) params.category = exportCategory;
      if (exportLanguage) params.language = exportLanguage;

      const jsonl = await exportBooks(params);
      const lines = jsonl.trim().split('\n').filter(Boolean).length;

      const blob = new Blob([jsonl], { type: 'application/x-ndjson' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'arkheion-export.jsonl';
      a.click();
      URL.revokeObjectURL(url);

      notify.success(`Exported ${lines} books`);
    } catch (err) {
      notify.error('Export failed: ' + err.message);
    } finally {
      exportLoading = false;
    }
  }
</script>

<svelte:head><title>Admin — Arkheion</title></svelte:head>

<div class="admin-page">
  <h1>Admin</h1>

  <!-- Webhooks -->
  <section class="section">
    <div class="section-header">
      <div>
        <h2>Webhooks</h2>
        <p class="text-muted text-sm">Push real-time events to Grimoire and other integrations.</p>
      </div>
      <button class="btn btn-primary" on:click={() => { showWebhookForm = true; webhookFormError = ''; }}>
        Add Webhook
      </button>
    </div>

    {#if webhooksLoading}
      <div style="padding:2rem; display:flex; justify-content:center"><div class="spinner"></div></div>
    {:else if webhooks.length === 0 && !showWebhookForm}
      <div class="empty-card">
        <p class="text-muted text-sm">No webhooks configured.</p>
        <p class="text-dim text-xs" style="margin-top:0.375rem">
          Add a webhook to notify Grimoire or other tools when books are added, updated, or extracted.
        </p>
      </div>
    {:else}
      <div class="webhooks-list">
        {#each webhooks as wh}
          <div class="webhook-item card">
            <div class="webhook-header">
              <div class="webhook-url">
                <span class="badge {wh.active ? 'badge-success' : 'badge-muted'}">
                  {wh.active ? 'Active' : 'Paused'}
                </span>
                <code class="url-text">{wh.url}</code>
              </div>
              <div class="webhook-actions">
                <button class="btn btn-secondary" style="font-size:0.75rem;padding:0.3rem 0.6rem"
                  on:click={() => toggleWebhook(wh)}>
                  {wh.active ? 'Pause' : 'Activate'}
                </button>
                <button class="btn btn-danger" style="font-size:0.75rem;padding:0.3rem 0.6rem"
                  on:click={() => handleDeleteWebhook(wh.id, wh.url)}>
                  Delete
                </button>
              </div>
            </div>
            {#if wh.description}
              <p class="text-muted text-sm" style="margin-top:0.25rem">{wh.description}</p>
            {/if}
            <div class="webhook-events">
              {#each wh.events as event}
                <span class="event-badge">{event}</span>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/if}

    {#if showWebhookForm}
      <div class="webhook-form card">
        <h3 style="margin-bottom:1.125rem">New Webhook</h3>

        {#if webhookFormError}
          <div class="alert alert-error">{webhookFormError}</div>
        {/if}

        <div class="field">
          <label class="label" for="wh-url">Endpoint URL *</label>
          <input id="wh-url" class="input" type="url"
            placeholder="https://your-tool.example.com/webhooks/arkheion"
            bind:value={webhookForm.url} />
        </div>

        <div class="field">
          <label class="label" for="wh-secret">Signing Secret * (min 16 chars)</label>
          <div class="secret-row">
            <input id="wh-secret" class="input" type="text"
              placeholder="At least 16 characters"
              bind:value={webhookForm.secret} />
            <button type="button" class="btn btn-secondary" on:click={generateSecret}>
              Generate
            </button>
          </div>
          <p class="text-xs text-dim" style="margin-top:0.375rem">
            Signs payloads with HMAC-SHA256. Verify the <code>X-Arkheion-Signature</code> header on your receiver.
          </p>
        </div>

        <div class="field">
          <label class="label">Events *</label>
          <div class="events-checkboxes">
            {#each allEvents as event}
              <label class="event-checkbox">
                <input type="checkbox"
                  checked={webhookForm.events.includes(event)}
                  on:change={() => toggleEvent(event)} />
                <code>{event}</code>
              </label>
            {/each}
          </div>
        </div>

        <div class="field">
          <label class="label" for="wh-desc">Description (optional)</label>
          <input id="wh-desc" class="input" type="text"
            placeholder="e.g. Grimoire knowledge graph sync"
            bind:value={webhookForm.description} />
        </div>

        <div class="form-actions">
          <button class="btn btn-secondary" on:click={() => showWebhookForm = false}>Cancel</button>
          <button class="btn btn-primary" on:click={handleCreateWebhook} disabled={webhookSubmitting}>
            {webhookSubmitting ? 'Creating…' : 'Create Webhook'}
          </button>
        </div>
      </div>
    {/if}
  </section>

  <!-- Export -->
  <section class="section">
    <div class="section-header">
      <div>
        <h2>Bulk Text Export</h2>
        <p class="text-muted text-sm">
          Export extracted book text as JSONL for LLM pipelines (Golem, fine-tuning).
          Only books with extracted text are included.
        </p>
      </div>
    </div>

    <div class="export-card card">
      <div class="export-filters">
        <div class="field">
          <label class="label" for="ex-tag">Tag filter</label>
          <input id="ex-tag" class="input" type="text" placeholder="e.g. philosophy"
            bind:value={exportTag} />
        </div>
        <div class="field">
          <label class="label" for="ex-cat">Category filter</label>
          <input id="ex-cat" class="input" type="text" placeholder="e.g. Science"
            bind:value={exportCategory} />
        </div>
        <div class="field">
          <label class="label" for="ex-lang">Language filter</label>
          <input id="ex-lang" class="input" type="text" placeholder="e.g. en"
            bind:value={exportLanguage} />
        </div>
      </div>

      <div class="export-actions">
        <button class="btn btn-primary" on:click={handleExport} disabled={exportLoading}>
          {exportLoading ? 'Exporting…' : 'Download JSONL'}
        </button>
      </div>

      <div class="export-info">
        <p class="info-label">JSONL format</p>
        <pre class="code-block">{"{"}"id":"uuid","title":"Book Title","authors":["Author Name"],"categories":["Science"],"language":"en","text":"Full extracted text..."{"}"}</pre>
        <p class="text-xs text-dim" style="margin-top:0.5rem">
          Each line is a self-contained JSON object. Compatible with Hugging Face datasets,
          Golem training pipelines, and most LLM fine-tuning frameworks.
        </p>
      </div>
    </div>
  </section>

  <!-- API Reference -->
  <section class="section">
    <h2>API Reference</h2>
    <p class="text-muted text-sm" style="margin-bottom:1rem">
      Arkheion exposes a full REST API. All requests require the <code>X-API-Key</code> header.
    </p>
    <div class="api-links">
      <a href="/api/v1/docs/openapi.yaml" target="_blank" class="btn btn-secondary">OpenAPI Spec</a>
      <a href="/api/v1/health" target="_blank" class="btn btn-secondary">Health Check</a>
    </div>
    <div class="api-example">
      <p class="info-label" style="margin-bottom:0.5rem">Example</p>
      <pre class="code-block">curl "http://localhost:8080/api/v1/books" \
  -H "X-API-Key: your-api-key" | jq .</pre>
    </div>
  </section>
</div>

<style>
  .admin-page { max-width: 820px; }
  h1 { margin-bottom: 1.75rem; }

  .section { margin-bottom: 2.75rem; }

  .section-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .section-header h2 { margin-bottom: 0.2rem; }

  .empty-card {
    background: var(--color-bg-card);
    border: 1px dashed var(--color-border-strong);
    border-radius: var(--radius-lg);
    padding: 1.75rem;
    text-align: center;
  }

  .webhooks-list { display: flex; flex-direction: column; gap: 0.625rem; margin-bottom: 0.875rem; }

  .webhook-item.card { padding: 0.875rem 1rem; }
  .webhook-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.875rem;
    margin-bottom: 0.375rem;
    flex-wrap: wrap;
  }
  .webhook-url { display: flex; align-items: center; gap: 0.625rem; min-width: 0; flex: 1; }
  .url-text {
    font-family: var(--font-mono);
    font-size: 0.775rem;
    color: var(--color-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .webhook-actions { display: flex; gap: 0.375rem; flex-shrink: 0; }
  .webhook-events { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-top: 0.4rem; }
  .event-badge {
    font-family: var(--font-mono);
    font-size: 0.675rem;
    padding: 0.1rem 0.375rem;
    background: rgba(192, 57, 43, 0.08);
    border: 1px solid rgba(192, 57, 43, 0.2);
    border-radius: 3px;
    color: var(--color-accent);
  }

  .webhook-form.card { padding: 1.375rem; margin-top: 0.875rem; }
  .secret-row { display: flex; gap: 0.625rem; }
  .events-checkboxes { display: flex; flex-direction: column; gap: 0.4rem; }
  .event-checkbox {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    cursor: pointer;
    font-size: 0.825rem;
  }
  .event-checkbox input { cursor: pointer; accent-color: var(--color-primary); }
  .event-checkbox code {
    font-family: var(--font-mono);
    font-size: 0.775rem;
    color: var(--color-text-muted);
  }
  .form-actions { display: flex; justify-content: flex-end; gap: 0.625rem; margin-top: 1.25rem; }

  /* Export */
  .export-card.card { padding: 1.375rem; }
  .export-filters {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 0 0.875rem;
    margin-bottom: 1rem;
  }
  .export-actions { margin-bottom: 1.25rem; }
  .export-info {
    background: var(--color-bg-elevated);
    border-radius: var(--radius);
    padding: 0.875rem;
    border: 1px solid var(--color-border);
  }
  .info-label {
    font-size: 0.675rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--color-text-dim);
    margin-bottom: 0.4rem;
  }

  /* API */
  .api-links { display: flex; gap: 0.625rem; margin-bottom: 1rem; flex-wrap: wrap; }
  .api-example {
    background: var(--color-bg-card);
    border-radius: var(--radius);
    padding: 0.875rem;
    border: 1px solid var(--color-border);
  }

  .code-block {
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    padding: 0.625rem 0.875rem;
    font-family: var(--font-mono);
    font-size: 0.725rem;
    color: var(--color-text-muted);
    overflow-x: auto;
    white-space: pre;
    line-height: 1.6;
  }

  code {
    font-family: var(--font-mono);
    font-size: 0.775rem;
    background: var(--color-bg-elevated);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    color: var(--color-text-muted);
    border: 1px solid var(--color-border);
  }
</style>
