(function () {
  'use strict';

  angular
    .module('fotoBrowser', [])
    .controller('AppCtrl', AppCtrl);

  function AppCtrl($scope, $location, $http) {
    $scope.location = $location;
    $scope.$watch('location.path()', function(path) {
      $scope.path = path || '/photos/';
      $http.get($scope.path).success(function(data) {
        $scope.files = data;
      });
    });
  }

})();
