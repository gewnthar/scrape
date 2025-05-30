// frontend/js/app.js
document.addEventListener('DOMContentLoaded', () => {
    console.log('FAA DST Frontend Initialized');

    // --- DOM Elements ---
    // (Keep all existing DOM Element selections as before)
    const findRoutesBtn = document.getElementById('find-routes-btn');
    const originInput = document.getElementById('origin');
    const destinationInput = document.getElementById('destination');
    const routeDateInput = document.getElementById('route-date');
    const routesResultsArea = document.getElementById('routes-results-area');

    const searchAdvisoriesBtn = document.getElementById('search-advisories-btn');
    const advisoryDateInput = document.getElementById('advisory-date');
    const advisoryKeywordInput = document.getElementById('advisory-keyword');
    const advisoriesSummaryArea = document.getElementById('advisories-summary-area');
    const advisoryDetailArea = document.getElementById('advisory-detail-area');
    
    const exploratorySearchBtn = document.getElementById('exploratory-search-btn');
    const exploratoryDateInput = document.getElementById('exploratory-date');
    const exploratoryStatusArea = document.getElementById('exploratory-status-area');

    const refreshCdrBtn = document.getElementById('refresh-cdr-btn');
    const cdrRefreshStatus = document.getElementById('cdr-refresh-status');
    const refreshPrefroutesBtn = document.getElementById('refresh-prefroutes-btn');
    const prefroutesRefreshStatus = document.getElementById('prefroutes-refresh-status');
    
    const checkUpdateCdrBtn = document.getElementById('check-update-cdr-btn');
    const cdrCheckStatus = document.getElementById('cdr-check-status');
    const checkUpdatePrefroutesBtn = document.getElementById('check-update-prefroutes-btn');
    const prefroutesCheckStatus = document.getElementById('prefroutes-check-status');

    // New elements for data status
    const cdrLastUpdatedSpan = document.getElementById('cdr-last-updated');
    const prefroutesLastUpdatedSpan = document.getElementById('prefroutes-last-updated');
    const currentYearSpan = document.getElementById('current-year');
    
    const API_BASE_URL = '/api'; 

    // --- Helper Functions ---
    // (displayResults and updateStatus helpers remain the same as before)
    function displayResults(areaElement, data, type) {
        if (typeof data === 'string') {
             areaElement.innerHTML = `<p>${data}</p>`;
             return;
        }
        areaElement.innerHTML = `<pre>${JSON.stringify(data, null, 2)}</pre>`;
    }

    function updateStatus(element, message, isError = false) {
        if (element) {
            element.textContent = message;
            element.style.color = isError ? 'var(--aviation-accent-orange)' : 'var(--aviation-accent-yellow)';
        }
    }

    // --- NEW: Function to Fetch and Display Data Source Status ---
    async function fetchAndDisplayDataStatus() {
        // Default messages
        if(cdrLastUpdatedSpan) cdrLastUpdatedSpan.textContent = 'N/A';
        if(prefroutesLastUpdatedSpan) prefroutesLastUpdatedSpan.textContent = 'N/A';

        try {
            const response = await fetch(`${API_BASE_URL}/admin/data-status`); // NEW API ENDPOINT
            const dataSources = await response.json();

            if (!response.ok) {
                throw new Error(dataSources.error || `HTTP error! status: ${response.status}`);
            }

            dataSources.forEach(source => {
                let displayText = 'Never updated';
                if (source.last_successfully_downloaded_at) {
                    displayText = `Downloaded: ${new Date(source.last_successfully_downloaded_at).toLocaleString()}`;
                    if (source.effective_until) {
                        displayText += ` (Effective until: ${new Date(source.effective_until).toLocaleDateString()})`;
                    }
                } else if (source.effective_until) {
                     displayText = `Effective until: ${new Date(source.effective_until).toLocaleDateString()}`;
                }


                if (source.source_name === 'CDR_CSV' && cdrLastUpdatedSpan) {
                    cdrLastUpdatedSpan.textContent = displayText;
                } else if (source.source_name === 'PREFERRED_ROUTES_CSV' && prefroutesLastUpdatedSpan) {
                    prefroutesLastUpdatedSpan.textContent = displayText;
                }
            });

        } catch (error) {
            console.error('Error fetching data source status:', error);
            if(cdrLastUpdatedSpan) cdrLastUpdatedSpan.textContent = 'Error fetching status';
            if(prefroutesLastUpdatedSpan) prefroutesLastUpdatedSpan.textContent = 'Error fetching status';
        }
    }

    // --- Event Listeners ---
    // (Find Routes - no change needed, backend defaults date if routeDateInput.value is empty)
    if (findRoutesBtn) {
        findRoutesBtn.addEventListener('click', async () => {
            const origin = originInput.value.trim().toUpperCase();
            const destination = destinationInput.value.trim().toUpperCase();
            const date = routeDateInput.value; // If empty, backend defaults to today

            if (!origin || !destination ) { // Date is now optional on frontend input
                displayResults(routesResultsArea, 'Origin and Destination are required.', 'error');
                return;
            }
            routesResultsArea.innerHTML = '<p>Finding routes...</p>';
            try {
                const payload = { origin, destination };
                if (date) { // Only include date in payload if user selected one
                    payload.date = date;
                }
                const response = await fetch(`${API_BASE_URL}/routes/find`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
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

    // (Search Advisories, Exploratory Search, Admin Refresh/Check buttons - listeners remain the same)
    // ... (keep all other existing event listeners from the previous app.js version) ...
    // Admin: Refresh CDRs
    if (refreshCdrBtn) {
        refreshCdrBtn.addEventListener('click', async () => {
            updateStatus(cdrRefreshStatus, 'Refreshing CDRs...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/refresh-routes/cdr`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(cdrRefreshStatus, data.message || 'CDR refresh initiated.');
                fetchAndDisplayDataStatus(); // Refresh status after action
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
                fetchAndDisplayDataStatus(); // Refresh status after action
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
                fetchAndDisplayDataStatus(); // Refresh status after action
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
                fetchAndDisplayDataStatus(); // Refresh status after action
            } catch (error) {
                console.error('Error checking/updating Preferred Routes:', error);
                updateStatus(prefroutesCheckStatus, `Error: ${error.message}`, true);
            }
        });
    }
    
    // Search Advisories (Summaries) - (Copied from previous version)
    if (searchAdvisoriesBtn) {
        searchAdvisoriesBtn.addEventListener('click', async () => {
            const date = advisoryDateInput.value;
            const keyword = advisoryKeywordInput.value.trim();

            if (!date) {
                displayResults(advisoriesSummaryArea, 'Date is required for advisory search.', 'error');
                return;
            }
            advisoriesSummaryArea.innerHTML = '<p>Searching advisories...</p>';
            advisoryDetailArea.innerHTML = ''; 

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
            } catch (error) {
                console.error('Error searching advisories:', error);
                displayResults(advisoriesSummaryArea, `Error searching advisories: ${error.message}`, 'error');
            }
        });
    }
    
    // Exploratory Search - (Copied from previous version)
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


    // --- Initial Page Load Actions ---
    if(currentYearSpan) {
        currentYearSpan.textContent = new Date().getFullYear();
    }
    fetchAndDisplayDataStatus(); // Fetch status on page load

});