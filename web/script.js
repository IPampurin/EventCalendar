// API-адреса
const API = {
    create: '/create_event',
    update: '/update_event',
    delete: '/delete_event',
    eventsForMonth: '/events_for_month',
    archive: '/archive_events'
};

let autoRefreshTimer = null;
const REFRESH_INTERVAL_MS = 5000;

function formatDateTime(dateStr) {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return isNaN(d.getTime()) ? dateStr : d.toLocaleString();
}

function formatDateInput(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '';
    return d.toISOString().slice(0, 16);
}

function shortenUid(uid) {
    if (!uid) return '-';
    return uid.substring(0, 8) + '…';
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        alert('UUID скопирован: ' + text);
    }).catch(() => {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        alert('UUID скопирован');
    });
}

function resetRefreshTimer() {
    if (autoRefreshTimer) clearInterval(autoRefreshTimer);
    autoRefreshTimer = setInterval(() => {
        loadEvents();
    }, REFRESH_INTERVAL_MS);
}

// Возвращает локальную дату в формате YYYY-MM-DD
function getCurrentDate() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}

// Загрузка активных событий за текущий месяц
async function loadEvents() {
    const userId = document.getElementById('user_id').value;
    const date = getCurrentDate();
    const url = `${API.eventsForMonth}?user_id=${userId}&date=${date}`;

    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const data = await response.json();
        let events = [];
        if (data.result && Array.isArray(data.result)) {
            events = data.result;
        } else if (Array.isArray(data)) {
            events = data;
        } else {
            console.warn('Неизвестный формат ответа активных событий:', data);
        }
        renderEventsTable(events);
    } catch (error) {
        console.error('Ошибка загрузки событий:', error);
        document.getElementById('eventsBody').innerHTML = '<tr><td colspan="8" style="text-align:center;">Ошибка загрузки событий</td></tr>';
    }
}

// Отрисовка таблицы активных событий
function renderEventsTable(events) {
    const tbody = document.getElementById('eventsBody');
    if (!events.length) {
        tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;">Нет событий за текущий месяц</td></tr>';
        return;
    }
    tbody.innerHTML = events.map(e => {
        const start = formatDateTime(e.start_at);
        const end = formatDateTime(e.end_at);
        const reminder = formatDateTime(e.reminder_at);
        const createdAt = formatDateTime(e.created_at);
        const shortId = shortenUid(e.id);
        const description = e.description || '';
        return `
            <tr data-id="${e.id}">
                <td>
                    <span class="uid-short" title="${e.id}" onclick="copyToClipboard('${e.id}')">${shortId}</span>
                    <button class="copy-btn" onclick="copyToClipboard('${e.id}')" title="Скопировать UUID">📋</button>
                </td>
                <td>${escapeHtml(e.title)}</td>
                <td title="${escapeHtml(description)}">${truncate(description, 30)}</td>
                <td>${start}</td>
                <td>${end}</td>
                <td>${reminder}</td>
                <td>${createdAt}</td>
                <td class="actions">
                    <button class="edit-btn" onclick="editEvent('${e.id}')">✏️</button>
                    <button class="delete-btn" onclick="deleteEventUI('${e.id}')">🗑️</button>
                </td>
            </tr>
        `;
    }).join('');
}

// Загрузка архивных событий
async function loadArchive() {
    const userId = document.getElementById('user_id').value;
    const url = `${API.archive}?user_id=${userId}&limit=100&offset=0`;
    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const data = await response.json();
        let events = [];
        if (data.result && Array.isArray(data.result)) {
            events = data.result;
        } else if (Array.isArray(data)) {
            events = data;
        } else {
            console.warn('Неизвестный формат ответа архива:', data);
        }
        renderArchiveTable(events);
    } catch (error) {
        console.error('Ошибка загрузки архива:', error);
        document.getElementById('archiveBody').innerHTML = '<tr><td colspan="7" style="text-align:center;">Ошибка загрузки архива</td></tr>';
    }
}

// Отрисовка таблицы архива
function renderArchiveTable(events) {
    const tbody = document.getElementById('archiveBody');
    if (!events.length) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;">Архив пуст</td></tr>';
        return;
    }
    tbody.innerHTML = events.map(e => {
        const start = formatDateTime(e.start_at);
        const end = formatDateTime(e.end_at);
        const reminder = formatDateTime(e.reminder_at);
        const archived = formatDateTime(e.archived_at);
        const description = e.description || '';
        return `
            <tr>
                <td title="${e.id}">${shortenUid(e.id)}</td>
                <td>${escapeHtml(e.title)}</td>
                <td title="${escapeHtml(description)}">${truncate(description, 30)}</td>
                <td>${start}</td>
                <td>${end}</td>
                <td>${reminder}</td>
                <td>${archived}</td>
            </tr>
        `;
    }).join('');
}

// Создание или обновление события
document.getElementById('eventForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const eventId = document.getElementById('event_id').value;
    const userId = parseInt(document.getElementById('user_id').value, 10);
    const title = document.getElementById('title').value.trim();
    const description = document.getElementById('description').value.trim();
    const startAtLocal = document.getElementById('start_at').value;
    const endAtLocal = document.getElementById('end_at').value;
    const reminderAtLocal = document.getElementById('reminder_at').value;

    if (!title) { alert('Название обязательно'); return; }
    if (!startAtLocal) { alert('Время начала обязательно'); return; }

    const startAt = new Date(startAtLocal).toISOString();
    const endAt = endAtLocal ? new Date(endAtLocal).toISOString() : null;
    const reminderAt = reminderAtLocal ? new Date(reminderAtLocal).toISOString() : null;

    let url = API.create;
    let method = 'POST';
    let payload = { user_id: userId, title, description, start_at: startAt, end_at: endAt, reminder_at: reminderAt };
    if (eventId) {
        url = API.update;
        payload.event_id = eventId;
    }

    try {
        const response = await fetch(url, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (!response.ok) {
            const errData = await response.json().catch(() => ({}));
            throw new Error(errData.error || `HTTP ${response.status}`);
        }
        resetForm();
        resetRefreshTimer();
        await loadEvents();
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
});

// Редактирование события
async function editEvent(id) {
    const userId = document.getElementById('user_id').value;
    const date = getCurrentDate();
    const url = `${API.eventsForMonth}?user_id=${userId}&date=${date}`;
    try {
        const response = await fetch(url);
        if (!response.ok) throw new Error('Не удалось загрузить события');
        const data = await response.json();
        let events = data.result || (Array.isArray(data) ? data : []);
        const event = events.find(e => e.id === id);
        if (!event) { alert('Событие не найдено'); return; }
        document.getElementById('event_id').value = event.id;
        document.getElementById('user_id').value = event.user_id;
        document.getElementById('title').value = event.title;
        document.getElementById('description').value = event.description || '';
        document.getElementById('start_at').value = formatDateInput(event.start_at);
        document.getElementById('end_at').value = formatDateInput(event.end_at);
        document.getElementById('reminder_at').value = formatDateInput(event.reminder_at);
        document.getElementById('submitBtn').textContent = '✏️ Обновить';
        document.getElementById('cancelEditBtn').style.display = 'inline-block';
    } catch (error) {
        alert('Ошибка загрузки данных для редактирования: ' + error.message);
    }
}

function cancelEdit() { resetForm(); }

function resetForm() {
    document.getElementById('event_id').value = '';
    document.getElementById('title').value = '';
    document.getElementById('description').value = '';
    document.getElementById('start_at').value = '';
    document.getElementById('end_at').value = '';
    document.getElementById('reminder_at').value = '';
    document.getElementById('submitBtn').textContent = '✚ Создать';
    document.getElementById('cancelEditBtn').style.display = 'none';
}

// Удаление события
async function deleteEventUI(id) {
    if (!confirm('Удалить событие?')) return;
    const userId = parseInt(document.getElementById('user_id').value, 10);
    try {
        const response = await fetch(API.delete, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ user_id: userId, event_id: id })
        });
        if (!response.ok) {
            const errData = await response.json().catch(() => ({}));
            throw new Error(errData.error || `HTTP ${response.status}`);
        }
        resetRefreshTimer();
        await loadEvents();
    } catch (error) {
        alert('Ошибка удаления: ' + error.message);
    }
}

// Показать/скрыть архив
document.getElementById('showArchivedBtn').addEventListener('click', async () => {
    const container = document.getElementById('archiveContainer');
    if (container.style.display === 'none') {
        await loadArchive();
        container.style.display = 'block';
    } else {
        container.style.display = 'none';
    }
});
document.getElementById('hideArchiveBtn')?.addEventListener('click', () => {
    document.getElementById('archiveContainer').style.display = 'none';
});
document.getElementById('cancelEditBtn').addEventListener('click', cancelEdit);

// Инициализация
document.addEventListener('DOMContentLoaded', () => {
    loadEvents();
    resetRefreshTimer();
    window.editEvent = editEvent;
    window.deleteEventUI = deleteEventUI;
    window.copyToClipboard = copyToClipboard;
});

// Вспомогательные функции
function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}

function truncate(str, len) {
    if (!str) return '';
    return str.length > len ? str.substring(0, len) + '…' : str;
}