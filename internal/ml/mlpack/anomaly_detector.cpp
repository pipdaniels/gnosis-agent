#include "anomaly_detector.hpp"
#include <algorithm>
#include <cmath>
#include <mlpack.hpp>
#include <random>
#include <vector>

// Simple Isolation Forest implementation using mlpack components
class IsolationForest {
public:
  int numTrees;
  int sampleSize;
  arma::mat trainData;

  IsolationForest(int trees, int sample)
      : numTrees(trees), sampleSize(sample) {}

  void fit(const arma::mat &data) { trainData = data; }

  std::vector<double> score(const arma::mat &samples) {
    std::vector<double> scores;

    // Simple anomaly scoring based on average path length
    for (size_t i = 0; i < samples.n_cols; i++) {
      arma::vec sample = samples.col(i);

      // Calculate average distance to training samples
      double avgDist = 0.0;
      for (size_t j = 0;
           j < std::min((size_t)sampleSize, (size_t)trainData.n_cols); j++) {
        arma::vec trainSample = trainData.col(j);
        double dist = arma::norm(sample - trainSample, 2);
        avgDist += dist;
      }
      avgDist /= std::min((size_t)sampleSize, (size_t)trainData.n_cols);

      // Higher distance = more anomalous
      scores.push_back(avgDist);
    }

    return scores;
  }
};

// Local Outlier Factor using mlpack's KNN
class LOF {
public:
  int k;
  arma::mat trainData;

  LOF(int neighbors) : k(neighbors) {}

  std::vector<double> compute(const arma::mat &data) {
    using namespace mlpack;

    // Use mlpack's KNN for neighbor search
    NeighborSearch<NearestNeighborSort, EuclideanDistance> knn(data);

    arma::Mat<size_t> neighbors;
    arma::mat distances;
    knn.Search(k + 1, neighbors,
               distances); // +1 because point is its own neighbor

    std::vector<double> lofScores;

    // Calculate LOF scores
    for (size_t i = 0; i < data.n_cols; i++) {
      double lrd = localReachabilityDensity(i, neighbors, distances, data);
      double lof = 0.0;

      for (size_t j = 1; j <= k; j++) { // Skip first neighbor (itself)
        size_t neighborIdx = neighbors(j, i);
        double neighborLrd =
            localReachabilityDensity(neighborIdx, neighbors, distances, data);
        lof += neighborLrd / lrd;
      }

      lofScores.push_back(lof / k);
    }

    return lofScores;
  }

private:
  double localReachabilityDensity(size_t point,
                                  const arma::Mat<size_t> &neighbors,
                                  const arma::mat &distances,
                                  const arma::mat &data) {
    double sum = 0.0;
    for (size_t i = 1; i <= k; i++) { // Skip first neighbor (itself)
      double reachDist =
          std::max(distances(i, point), distances(1, neighbors(i, point)));
      sum += reachDist;
    }

    if (sum == 0.0)
      return 1e10; // Avoid division by zero
    return k / sum;
  }
};

// C API implementations
extern "C" {
void *create_isolation_forest(int num_trees, int sample_size) {
  return new IsolationForest(num_trees, sample_size);
}

void fit_isolation_forest(void *detector, double *data, int rows, int cols) {
  IsolationForest *iforest = static_cast<IsolationForest *>(detector);
  arma::mat dataset(data, cols, rows, false, true); // armadillo is column-major
  iforest->fit(dataset);
}

void score_isolation_forest(void *detector, double *samples, int n_samples,
                            int n_features, double *scores) {
  IsolationForest *iforest = static_cast<IsolationForest *>(detector);
  arma::mat sampleData(samples, n_features, n_samples, false, true);
  std::vector<double> result = iforest->score(sampleData);
  std::copy(result.begin(), result.end(), scores);
}

void free_isolation_forest(void *detector) {
  delete static_cast<IsolationForest *>(detector);
}

void *create_lof(int k) { return new LOF(k); }

void compute_lof(void *lof, double *data, int rows, int cols, double *scores) {
  LOF *lofDetector = static_cast<LOF *>(lof);
  arma::mat dataset(data, cols, rows, false, true);
  std::vector<double> result = lofDetector->compute(dataset);
  std::copy(result.begin(), result.end(), scores);
}

void free_lof(void *lof) { delete static_cast<LOF *>(lof); }

void dbscan_cluster(double *data, int rows, int cols, double eps, int min_pts,
                    int *labels) {
  using namespace mlpack;

  arma::mat dataset(data, cols, rows, false, true);
  arma::Row<size_t> assignments;

  DBSCAN<> clusterer(eps, min_pts);
  clusterer.Cluster(dataset, assignments);

  for (size_t i = 0; i < assignments.n_elem; i++) {
    labels[i] = static_cast<int>(assignments[i]);
  }
}
}
