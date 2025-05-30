// frontend/js/app.js
document.addEventListener('DOMContentLoaded', () => {
    console.log('FAA DST Frontend Initialized');

    // --- DOM Elements ---
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

    const cdrLastUpdatedSpan = document.getElementById('cdr-last-updated');
    const prefroutesLastUpdatedSpan = document.getElementById('prefroutes-last-updated');
    const currentYearSpan = document.getElementById('current-year');
    
    const API_BASE_URL = '/api'; 

    // --- Helper Functions ---
    function displayResults(areaElement, data, type) {
        areaElement.innerHTML = ''; // Clear previous results

        if (type === 'error' || typeof data === 'string') {
            const p = document.createElement('p');
            p.textContent = data;
            if (type === 'error') p.style.color = 'var(--aviation-accent-orange)';
            areaElement.appendChild(p);
            return;
        }

        if (type === 'status') { 
            const p = document.createElement('p');
            p.textContent = data.message || JSON.stringify(data);
            areaElement.appendChild(p);
            return;
        }

        if (type === 'routes' && Array.isArray(data)) {
            if (data.length === 0) {
                areaElement.innerHTML = '<p>No routes found matching your criteria.</p>';
                return;
            }

            const routeGroups = {
                preferred: [],
                cdrNoCoord: [],
                cdrCoordReq: [],
                rqdAdvisory: [],
                other: []
            };

            data.forEach(route => {
                if (route.Source && route.Source.includes("Preferred Route")) {
                    routeGroups.preferred.push(route);
                } else if (route.Source && route.Source.includes("CDR (No Coord)")) {
                    routeGroups.cdrNoCoord.push(route);
                } else if (route.Source && route.Source.includes("CDR (Coord Req)")) {
                    routeGroups.cdrCoordReq.push(route);
                } else if (route.Source && route.Source.includes("RQD Advisory")) {
                    routeGroups.rqdAdvisory.push(route);
                } 
                else {
                    routeGroups.other.push(route);
                }
            });

            const createRouteTable = (routes, title) => {
                if (routes.length === 0) return '';
                
                let tableHtml = `<h3>${title}</h3><table><thead><tr>
                                    <th>Origin</th>
                                    <th>Destination</th>
                                    <th>Route Code</th>
                                    <th>Departure Fix</th>
                                    <th>Route String</th>
                                    <th>Source</th>
                                    <th>Restrictions/Notes</th>
                                 </tr></thead><tbody>`;
                routes.forEach(r => {
                    const origin = r.Origin || (r.Cdr ? r.Cdr.Origin : (r.Preferred ? r.Preferred.Origin : 'N/A'));
                    const dest = r.Destination || (r.Cdr ? r.Cdr.Destination : (r.Preferred ? r.Preferred.Destination : 'N/A'));
                    const routeCode = r.Cdr ? (r.Cdr.RouteCode || 'N/A') : 'N/A'; // Handle if Cdr.RouteCode is empty
                    const depFix = r.Cdr ? (r.Cdr.DepartureFix || 'N/A') : 'N/A';
                    const restrictions = r.Restrictions || r.Justification || '';

                    tableHtml += `<tr>
                                    <td>${origin}</td>
                                    <td>${dest}</td>
                                    <td>${routeCode}</td>
                                    <td>${depFix}</td>
                                    <td>${r.RouteString || 'N/A'}</td>
                                    <td>${r.Source || 'N/A'}</td>
                                    <td>${restrictions}</td>
                                  </tr>`;
                });
                tableHtml += '</tbody></table>';
                return tableHtml;
            };

            let content = '';
            if(routeGroups.rqdAdvisory.length > 0) {
                 content += createRouteTable(routeGroups.rqdAdvisory, 'Routes from Mandatory Advisories');
            }
            content += createRouteTable(routeGroups.preferred, 'Preferred Routes');
            content += createRouteTable(routeGroups.cdrNoCoord, 'CDRs (No Coordination Required)');
            content += createRouteTable(routeGroups.cdrCoordReq, 'CDRs (Coordination Required)');
            if(routeGroups.other.length > 0) {
                 content += createRouteTable(routeGroups.other, 'Other Routes/Info');
            }

            areaElement.innerHTML = content;

        } else if (type === 'summaries' && Array.isArray(data)) {
            if (data.length === 0) {
                areaElement.innerHTML = '<p>No advisories found.</p>';
                return;
            }
            const ul = document.createElement('ul');
            ul.className = 'advisory-summary-list';
            data.forEach(summary => {
                const li = document.createElement('li');
                li.innerHTML = `<strong>${summary.SummaryUniqueKey || 'Advisory'}</strong>: ${summary.ListPageRawText || 'No raw text.'} 
                                <br><small>Has Detail Saved: ${summary.HasFullDetailSaved}</small>`;
                ul.appendChild(li);
            });
            areaElement.appendChild(ul);
        } else {
            areaElement.innerHTML = `<p>Received data (type: ${type}):</p><pre>${JSON.stringify(data, null, 2)}</pre>`;
        }
    }

    function updateStatus(element, message, isError = false) {
        if (element) {
            element.textContent = message;
            element.style.color = isError ? 'var(--aviation-accent-orange)' : 'var(--aviation-accent-yellow)';
        }
    }
    
    async function fetchAndDisplayDataStatus() {
        if(cdrLastUpdatedSpan) cdrLastUpdatedSpan.textContent = 'N/A';
        if(prefroutesLastUpdatedSpan) prefroutesLastUpdatedSpan.textContent = 'N/A';

        try {
            const response = await fetch(`${API_BASE_URL}/admin/data-status`); 
            const dataSources = await response.json();

            if (!response.ok) {
                throw new Error(dataSources.error || `HTTP error! status: ${response.status}`);
            }

            dataSources.forEach(source => {
                let displayText = 'Never updated';
                if (source.last_successfully_downloaded_at) {
                    displayText = `Refreshed: ${new Date(source.last_successfully_downloaded_at).toLocaleString()}`;
                    if (source.effective_until) {
                        displayText += ` (Data effective until: ${new Date(source.effective_until).toLocaleDateString()})`;
                    } else {
                        displayText += ` (Data effective dates not specified by FAA for this batch)`;
                    }
                } else if (source.effective_until) {
                     displayText = `Data effective until: ${new Date(source.effective_until).toLocaleDateString()}`;
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
    if (findRoutesBtn) {
        findRoutesBtn.addEventListener('click', async () => {
            const origin = originInput.value.trim().toUpperCase();
            const destination = destinationInput.value.trim().toUpperCase();
            const date = routeDateInput.value; 

            if (!origin || !destination ) { 
                displayResults(routesResultsArea, 'Origin and Destination are required.', 'error');
                return;
            }
            routesResultsArea.innerHTML = '<p>Finding routes...</p>';
            try {
                const payload = { origin, destination };
                if (date) { 
                    payload.date = date;
                } // Backend defaults to today if date is not sent
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

    if (refreshCdrBtn) {
        refreshCdrBtn.addEventListener('click', async () => {
            updateStatus(cdrRefreshStatus, 'Refreshing CDRs...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/refresh-routes/cdr`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(cdrRefreshStatus, data.message || 'CDR refresh initiated.');
                fetchAndDisplayDataStatus(); 
            } catch (error) {
                console.error('Error refreshing CDRs:', error);
                updateStatus(cdrRefreshStatus, `Error: ${error.message}`, true);
            }
        });
    }

    if (refreshPrefroutesBtn) {
        refreshPrefroutesBtn.addEventListener('click', async () => {
            updateStatus(prefroutesRefreshStatus, 'Refreshing Preferred Routes...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/refresh-routes/preferredroutes`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(prefroutesRefreshStatus, data.message || 'Preferred Routes refresh initiated.');
                fetchAndDisplayDataStatus(); 
            } catch (error) {
                console.error('Error refreshing Preferred Routes:', error);
                updateStatus(prefroutesRefreshStatus, `Error: ${error.message}`, true);
            }
        });
    }
    
    if (checkUpdateCdrBtn) {
        checkUpdateCdrBtn.addEventListener('click', async () => {
            updateStatus(cdrCheckStatus, 'Checking/Updating CDRs...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/check-update-routes/cdr`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(cdrCheckStatus, data.message || 'CDR check/update process completed.');
                fetchAndDisplayDataStatus(); 
            } catch (error) {
                console.error('Error checking/updating CDRs:', error);
                updateStatus(cdrCheckStatus, `Error: ${error.message}`, true);
            }
        });
    }

    if (checkUpdatePrefroutesBtn) {
        checkUpdatePrefroutesBtn.addEventListener('click', async () => {
            updateStatus(prefroutesCheckStatus, 'Checking/Updating Preferred Routes...');
            try {
                const response = await fetch(`${API_BASE_URL}/admin/check-update-routes/preferredroutes`, { method: 'POST' });
                const data = await response.json();
                if (!response.ok) throw new Error(data.error || `HTTP error ${response.status}`);
                updateStatus(prefroutesCheckStatus, data.message || 'Preferred Routes check/update process completed.');
                fetchAndDisplayDataStatus(); 
            } catch (error) {
                console.error('Error checking/updating Preferred Routes:', error);
                updateStatus(prefroutesCheckStatus, `Error: ${error.message}`, true);
            }
        });
    }
    
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
    fetchAndDisplayDataStatus(); 

});