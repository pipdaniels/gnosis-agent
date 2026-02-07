#ifndef ANOMALY_DETECTOR_HPP
#define ANOMALY_DETECTOR_HPP

#ifdef __cplusplus
extern "C" {
#endif

// Isolation Forest
void *create_isolation_forest(int num_trees, int sample_size);
void fit_isolation_forest(void *detector, double *data, int rows, int cols);
void score_isolation_forest(void *detector, double *samples, int n_samples,
                            int n_features, double *scores);
void free_isolation_forest(void *detector);

// Local Outlier Factor
void *create_lof(int k);
void compute_lof(void *lof, double *data, int rows, int cols, double *scores);
void free_lof(void *lof);

// DBSCAN Clustering
void dbscan_cluster(double *data, int rows, int cols, double eps, int min_pts,
                    int *labels);

#ifdef __cplusplus
}
#endif

#endif
