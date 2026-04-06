/**
 * Shared utility functions for admin templates.
 */

/**
 * Renders pagination controls into the specified container element.
 * @param {string} containerId - ID of the div to render pagination into.
 * @param {object} meta - Object with { page, total_pages } from API response.
 * @param {object} [extraParams] - Optional key-value pairs for extra URL params.
 */
function renderPagination(containerId, meta, extraParams) {
    var container = document.getElementById(containerId);
    if (!container) return;
    if (!meta || meta.total_pages <= 1) return;

    var currentPage = meta.page;
    var totalPages = meta.total_pages;

    container.className = 'pagination';
    while (container.firstChild) {
        container.removeChild(container.firstChild);
    }

    function buildHref(page) {
        var params = 'page=' + page;
        if (extraParams) {
            for (var key in extraParams) {
                if (extraParams.hasOwnProperty(key) && extraParams[key] != null && extraParams[key] !== '') {
                    params += '&' + encodeURIComponent(key) + '=' + encodeURIComponent(extraParams[key]);
                }
            }
        }
        return '?' + params;
    }

    if (currentPage > 1) {
        var prev = document.createElement('a');
        prev.href = buildHref(currentPage - 1);
        prev.textContent = '\u2039 prev';
        container.appendChild(prev);
    }

    for (var i = 1; i <= totalPages; i++) {
        if (i === currentPage) {
            var span = document.createElement('span');
            span.className = 'active';
            span.textContent = i;
            container.appendChild(span);
        } else {
            var link = document.createElement('a');
            link.href = buildHref(i);
            link.textContent = i;
            container.appendChild(link);
        }
    }

    if (currentPage < totalPages) {
        var next = document.createElement('a');
        next.href = buildHref(currentPage + 1);
        next.textContent = 'next \u203A';
        container.appendChild(next);
    }
}

/**
 * Formats an ISO date string into YYYY-MM-DD HH:MM.
 * @param {string} isoString - ISO 8601 date string.
 * @returns {string} Formatted date or em-dash if falsy.
 */
function formatDate(isoString) {
    if (!isoString) return '\u2014';
    // ISO format: 2024-01-15T09:30:00Z - slice to YYYY-MM-DD HH:MM
    return isoString.slice(0, 10) + ' ' + isoString.slice(11, 16);
}

/**
 * Returns a DOM element showing a truncated UUID.
 * @param {string} uuid - Full UUID string.
 * @param {string} [linkHref] - Optional href to make the element a link.
 * @returns {HTMLElement} An <a> or <span> element with truncated UUID.
 */
function truncateUUID(uuid, linkHref) {
    var short = uuid ? uuid.slice(0, 8) + '...' : '';
    var el;
    if (linkHref) {
        el = document.createElement('a');
        el.href = linkHref;
    } else {
        el = document.createElement('span');
    }
    el.className = 'uuid';
    el.title = uuid || '';
    el.textContent = short;
    return el;
}
