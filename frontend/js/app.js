// frontend/js/app.js
document.addEventListener('DOMContentLoaded', () => {
    console.log('FAA DST Frontend Initialized');

    // --- DOM Elements ---
    // Route Finder
    const findRoutesBtn = document.getElementById('find-routes-btn');
    const originInput = document.getElementById('origin');
    const destinationInput = document.getElementById('destination');
    const routeDateInput = document.getElementById('route-date');
    const routesResultsArea = document.getElementById('routes-results-area');

    // Advisory Search
    const searchAdvisoriesBtn = document.getElementById('search-advisories-btn');
    const advisoryDateInput = document.getElementById('advisory-date');
    const advisoryKeywordInput = document.getElementById('advisory-keyword');
    const advisoriesSummaryArea = document.getElementById('advisories-summary-area');
    const advisoryDetailArea = document.getElementById('advisory-detail-area');

    // Exploratory Search
    const exploratorySearchBtn = document.getElementById('exploratory-search-btn');
    const exploratoryDateInput = document.getElementById('exploratory-date');
    const exploratoryStatusArea = document.getElementById('exploratory-status-area');

    // Admin Actions
    const refreshCdrBtn = document.getElementById('refresh-cdr-btn');
    const cdrRefreshStatus = document.getElementById('cdr-refresh-status');
    const refreshPrefroutesBtn = document.getElementById('refresh-prefroutes-btn');
    const prefroutesRefreshStatus = document.getElementById('prefroutes-refresh-status');
    const checkUpdateCdrBtn = document.getElementById('check-update-cdr-btn');
    const cdrCheckStatus = document.getElementById('cdr-check-status');
    const checkUpdatePrefroutesBtn = document.getElementById('check-update-prefroutes-btn');
    const prefroutesCheckStatus = document.getElementById('prefroutes-check-status');
    
    const API_BASE_URL = '/api'; // Assuming your Go API is served under /api by Apache proxy

    // --- Helper Functions ---
    function displayResults(areaElement, data, type) {
        if (typeof data === 'string') { // For simple messages
             areaElement.innerHTML = `<p>${data}</p>`;
             return;
        }

        // For actual data, pretty print JSON for now
        areaElement.innerHTML = `<pre>${JSON.stringify(data, null, 2)}</pre>`;
        // TODO: Implement proper rendering for each type of data
        // e.g., create tables for routes, lists for summaries, etc.
    }

    function updateStatus(element, message, isError = false) {
        element.textContent = message;
        element.style.color = isError ? 'var(--aviation-accent-orange)' : 'var(--aviation-accent-yellow)';
    }


    // --- Event Listeners ---

    // Find Routes
    if (findRoutesBtn) {
        findRoutesBtn.addEventListener('click', async () => {
            const origin = originInput.value.trim().toUpperCase();
            const destination = destinationInput.value.trim().toUpperCase();
            const date = routeDateInput.value;

            if (!origin || !destination || !date) {
                displayResults(routesResultsArea, 'Origin, Destination, and Date are required.', 'error');
                return;
            }
            routesResultsArea.innerHTML = '<p>Finding routes...</p>';
            try {
                const response = await fetch(`${API_BASE_URL}/routes/find`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ origin, destination, date })
                });
                const data = await response.json();
                if (!response.ok) {
                    throw new Error(data.error || `HTTP error! status: ${response.status}`);
                }
                displayResults(routesResultsArea, data, 'routes');
            } catch (error) {
                console.error('Error finding routes:', error);
                displayResults(routesResultsArea, `Error finding routes: ${error.message}`, 'error');
            }
        });
    }

    // Search Advisories (Summaries)
    if (searchAdvisoriesBtn) {
        searchAdvisoriesBtn.addEventListener('click', async () => {
            const date = advisoryDateInput.value;
            const keyword = advisoryKeywordInput.value.trim();

            if (!date) {
                displayResults(advisoriesSummaryArea, 'Date is required for advisory search.', 'error');
                return;
            }
            advisoriesSummaryArea.innerHTML = '<p>Searching advisories...</p>';
            advisoryDetailArea.innerHTML = ''; // Clear detail area

            let url = `${API_BASE_URL}/advisories?date=${date}`;
            if (keyword) {
                url += `&keyword=${encodeURIComponent(keyword)}`;
            }

            try {
                const response = await fetch(url);
                const data = await response.json();
                 if (!response.ok) {
                    throw new Error(data.error || `HTTP error! status: ${response.status}`);
                }
                displayResults(advisoriesSummaryArea, data, 'summaries');
                // TODO: Add click listeners to summaries to fetch/display details
            } catch (error) {
                console.error('Error searching advisories:', error);
                displayResults(advisoriesSummaryArea, `Error searching advisories: ${error.message}`, 'error');
            }
        });
    }
    
    // TODO: Implement click listeners for advisory summaries that call GetAdvisoryDetailHandler
    // and then potentially ConfirmSaveAdvisoryDetailHandler

    // Exploratory Search
    if(exploratorySearchBtn) {
        exploratorySearchBtn.addEventListener('click', async () => {
            const date = exploratoryDateInput.value;
            if (!date) {
                displayResults(exploratoryStatusArea, 'Date is required for exploratory scrape.', 'error');
                return;
            }
            exploratoryStatusArea.innerHTML = '<p>Initiating exploratory scrape...</p>';
            try {
                const response = await fetch(`${API_BASE_URL}/advisories/exploratory-search`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ date })
                });
                const data = await response.json();
                if (!response.ok) {
                    throw new Error(data.error || `HTTP error! status: ${response.status}`);
                }
                displayResults(exploratoryStatusArea, data.message || 'Request accepted.', 'status');
            } catch (error) {
                console.error('Error initiating exploratory scrape:', error);
                displayResults(exploratoryStatusArea, `Error: ${error.message}`, 'error');
            }
        });
    }


    // Admin: Refresh CDRs
    if (refreshCdrBtn) {
        refreshCdrBtn.addEventListener('click', async () => {
            updateStatus(cdrRefreshStatus, 'Refreshing CDRs...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/refresh-routes/cdr`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(cdrRefreshStatus, data.message || 'CDR refresh initiated.');
            } catch (error) {
                console.error('Error refreshing CDRs:', error);
                updateStatus(cdrRefreshStatus, `Error: ${error.message}`, true);
            }
        });
    }

    // Admin: Refresh Preferred Routes
    if (refreshPrefroutesBtn) {
        refreshPrefroutesBtn.addEventListener('click', async () => {
            updateStatus(prefroutesRefreshStatus, 'Refreshing Preferred Routes...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/refresh-routes/preferredroutes`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(prefroutesRefreshStatus, data.message || 'Preferred Routes refresh initiated.');
            } catch (error) {
                console.error('Error refreshing Preferred Routes:', error);
                updateStatus(prefroutesRefreshStatus, `Error: ${error.message}`, true);
            }
        });
    }
    
    // Admin: Check & Update CDRs
    if (checkUpdateCdrBtn) {
        checkUpdateCdrBtn.addEventListener('click', async () => {
            updateStatus(cdrCheckStatus, 'Checking/Updating CDRs...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/check-update-routes/cdr`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(cdrCheckStatus, data.message || 'CDR check/update process completed.');
            } catch (error) {
                console.error('Error checking/updating CDRs:', error);
                updateStatus(cdrCheckStatus, `Error: ${error.message}`, true);
            }
        });
    }

    // Admin: Check & Update Preferred Routes
    if (checkUpdatePrefroutesBtn) {
        checkUpdatePrefroutesBtn.addEventListener('click', async () => {
            updateStatus(prefroutesCheckStatus, 'Checking/Updating Preferred Routes...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/check-update-routes/preferredroutes`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(prefroutesCheckStatus, data.message || 'Preferred Routes check/update process completed.');
            } catch (error) {
                console.error('Error checking/updating Preferred Routes:', error);
                updateStatus(prefroutesCheckStatus, `Error: ${error.message}`, true);
            }
        });
    }

});