// GNOSISAGENT - IndexedDB Wrapper for Offline Storage

const DB_NAME = 'GNOSISAGENT_db';
const DB_VERSION = 1;

class OfflineStorage {
    constructor() {
        this.db = null;
        this.initPromise = null;
    }

    async init() {
        if (this.initPromise) return this.initPromise;
        
        this.initPromise = new Promise((resolve, reject) => {
            const request = indexedDB.open(DB_NAME, DB_VERSION);

            request.onerror = () => {
                this.initPromise = null;
                reject(request.error);
            };
            request.onsuccess = () => {
                this.db = request.result;
                resolve(this.db);
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;
                if (!db.objectStoreNames.contains('datasets')) {
                    const datasetStore = db.createObjectStore('datasets', { keyPath: 'id' });
                    datasetStore.createIndex('orgId', 'org_id', { unique: false });
                    datasetStore.createIndex('uploadedAt', 'uploaded_at', { unique: false });
                }
                if (!db.objectStoreNames.contains('analyses')) {
                    const analysisStore = db.createObjectStore('analyses', { keyPath: 'id' });
                    analysisStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }
                if (!db.objectStoreNames.contains('decisions')) {
                    const decisionStore = db.createObjectStore('decisions', { keyPath: 'decision_id' });
                    decisionStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }
                if (!db.objectStoreNames.contains('uploadQueue')) {
                    db.createObjectStore('uploadQueue', { keyPath: 'id', autoIncrement: true });
                }
                if (!db.objectStoreNames.contains('reports')) {
                    const reportStore = db.createObjectStore('reports', { keyPath: 'report_id' });
                    reportStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }
            };
        });
        return this.initPromise;
    }

    // Helper to wrap IDBRequest in a Promise
    _request(request) {
        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Helper for transactions
    async _transaction(stores, mode, callback) {
        if (!this.db) await this.init();
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(stores, mode);
            tx.oncomplete = () => resolve();
            tx.onerror = () => reject(tx.error);
            callback(tx);
        });
    }

    async saveDataset(dataset) {
        await this._transaction(['datasets'], 'readwrite', tx => {
            tx.objectStore('datasets').put(dataset);
        });
    }

    async getDataset(id) {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['datasets'], 'readonly');
        return this._request(tx.objectStore('datasets').get(id));
    }

    async getAllDatasets() {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['datasets'], 'readonly');
        return this._request(tx.objectStore('datasets').getAll());
    }

    async saveAnalysis(analysis) {
        await this._transaction(['analyses'], 'readwrite', tx => {
            tx.objectStore('analyses').put(analysis);
        });
    }

    async getAnalysis(id) {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['analyses'], 'readonly');
        return this._request(tx.objectStore('analyses').get(id));
    }

    async saveDecisions(decisions) {
        await this._transaction(['decisions'], 'readwrite', tx => {
            const store = tx.objectStore('decisions');
            (decisions.decisions || []).forEach(d => store.put(d));
        });
    }

    async getDecisionsByDataset(datasetId) {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['decisions'], 'readonly');
        const index = tx.objectStore('decisions').index('datasetId');
        return this._request(index.getAll(datasetId));
    }

    async queueUpload(data) {
        await this._transaction(['uploadQueue'], 'readwrite', tx => {
            tx.objectStore('uploadQueue').add({
                data: data,
                timestamp: Date.now(),
                status: 'pending'
            });
        });
    }

    async getUploadQueue() {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['uploadQueue'], 'readonly');
        return this._request(tx.objectStore('uploadQueue').getAll());
    }

    async removeFromQueue(id) {
        await this._transaction(['uploadQueue'], 'readwrite', tx => {
            tx.objectStore('uploadQueue').delete(id);
        });
    }

    async syncQueued() {
        const queue = await this.getUploadQueue();
        const results = [];

        for (const item of (queue || [])) {
            try {
                const response = await fetch('/api/upload', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(item.data)
                });

                if (response.ok) {
                    await this.removeFromQueue(item.id);
                    results.push({ success: true, id: item.id });
                } else {
                    results.push({ success: false, id: item.id, error: await response.text() });
                }
            } catch (error) {
                results.push({ success: false, id: item.id, error: error.message });
            }
        }
        return results;
    }

    async saveReport(report) {
        await this._transaction(['reports'], 'readwrite', tx => {
            tx.objectStore('reports').put(report);
        });
    }

    async getReport(reportId) {
        if (!this.db) await this.init();
        const tx = this.db.transaction(['reports'], 'readonly');
        return this._request(tx.objectStore('reports').get(reportId));
    }

    async clearAll() {
        const stores = ['datasets', 'analyses', 'decisions', 'uploadQueue', 'reports'];
        await this._transaction(stores, 'readwrite', tx => {
            stores.forEach(s => tx.objectStore(s).clear());
        });
    }
}
}

// Initialize storage on page load
const storage = new OfflineStorage();
storage.init().catch(console.error);

// Export for use in other scripts
window.OfflineStorage = storage;
