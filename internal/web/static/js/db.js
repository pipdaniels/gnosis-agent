// GeoAgent - IndexedDB Wrapper for Offline Storage

const DB_NAME = 'geoagent_db';
const DB_VERSION = 1;

class OfflineStorage {
    constructor() {
        this.db = null;
    }

    async init() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(DB_NAME, DB_VERSION);

            request.onerror = () => reject(request.error);
            request.onsuccess = () => {
                this.db = request.result;
                resolve(this.db);
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;

                // Datasets store
                if (!db.objectStoreNames.contains('datasets')) {
                    const datasetStore = db.createObjectStore('datasets', { key Path: 'id' });
                    datasetStore.createIndex('orgId', 'org_id', { unique: false });
                    datasetStore.createIndex('uploadedAt', 'uploaded_at', { unique: false });
                }

                // Analyses store
                if (!db.objectStoreNames.contains('analyses')) {
                    const analysisStore = db.createObjectStore('analyses', { keyPath: 'id' });
                    analysisStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }

                // Decisions store
                if (!db.objectStoreNames.contains('decisions')) {
                    const decisionStore = db.createObjectStore('decisions', { keyPath: 'decision_id' });
                    decisionStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }

                // Upload queue for offline support
                if (!db.objectStoreNames.contains('uploadQueue')) {
                    db.createObjectStore('uploadQueue', { keyPath: 'id', autoIncrement: true });
                }

                // Reports store
                if (!db.objectStoreNames.contains('reports')) {
                    const reportStore = db.createObjectStore('reports', { keyPath: 'report_id' });
                    reportStore.createIndex('datasetId', 'dataset_id', { unique: false });
                }
            };
        });
    }

    async saveDataset(dataset) {
        const tx = this.db.transaction(['datasets'], 'readwrite');
        const store = tx.objectStore('datasets');
        await store.put(dataset);
        return tx.complete;
    }

    async getDataset(id) {
        const tx = this.db.transaction(['datasets'], 'readonly');
        const store = tx.objectStore('datasets');
        return await store.get(id);
    }

    async getAllDatasets() {
        const tx = this.db.transaction(['datasets'], 'readonly');
        const store = tx.objectStore('datasets');
        return await store.getAll();
    }

    async saveAnalysis(analysis) {
        const tx = this.db.transaction(['analyses'], 'readwrite');
        const store = tx.objectStore('analyses');
        await store.put(analysis);
        return tx.complete;
    }

    async getAnalysis(id) {
        const tx = this.db.transaction(['analyses'], 'readonly');
        const store = tx.objectStore('analyses');
        return await store.get(id);
    }

    async saveDecisions(decisions) {
        const tx = this.db.transaction(['decisions'], 'readwrite');
        const store = tx.objectStore('decisions');
        for (const decision of decisions.decisions) {
            await store.put(decision);
        }
        return tx.complete;
    }

    async getDecisionsByDataset(datasetId) {
        const tx = this.db.transaction(['decisions'], 'readonly');
        const store = tx.objectStore('decisions');
        const index = store.index('datasetId');
        return await index.getAll(datasetId);
    }

    async queueUpload(data) {
        const tx = this.db.transaction(['uploadQueue'], 'readwrite');
        const store = tx.objectStore('uploadQueue');
        const item = {
            data: data,
            timestamp: Date.now(),
            status: 'pending'
        };
        await store.add(item);
        return tx.complete;
    }

    async getUploadQueue() {
        const tx = this.db.transaction(['uploadQueue'], 'readonly');
        const store = tx.objectStore('uploadQueue');
        return await store.getAll();
    }

    async removeFromQueue(id) {
        const tx = this.db.transaction(['uploadQueue'], 'readwrite');
        const store = tx.objectStore('uploadQueue');
        await store.delete(id);
        return tx.complete;
    }

    async syncQueued() {
        const queue = await this.getUploadQueue();
        const results = [];

        for (const item of queue) {
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
        const tx = this.db.transaction(['reports'], 'readwrite');
        const store = tx.objectStore('reports');
        await store.put(report);
        return tx.complete;
    }

    async getReport(reportId) {
        const tx = this.db.transaction(['reports'], 'readonly');
        const store = tx.objectStore('reports');
        return await store.get(reportId);
    }

    async clearAll() {
        const stores = ['datasets', 'analyses', 'decisions', 'uploadQueue', 'reports'];
        const tx = this.db.transaction(stores, 'readwrite');

        for (const storeName of stores) {
            const store = tx.objectStore(storeName);
            await store.clear();
        }

        return tx.complete;
    }
}

// Initialize storage on page load
const storage = new OfflineStorage();
storage.init().catch(console.error);

// Export for use in other scripts
window.OfflineStorage = storage;
