angular.module('GingerApp', ['ui.bootstrap'], function ($interpolateProvider) {
  $interpolateProvider.startSymbol('[[');
  $interpolateProvider.endSymbol(']]');
});

function GingerCtrl($scope) {
    $scope.state = {};
    $scope.errors = [];
    $scope.connection = null;

    $scope.NewConnection = function() {
        connection = new WebSocket('ws://'+document.location.host+document.location.pathname+'state');

        connection.onopen = function () {
        };

        connection.onclose = function (e) {
        };

        connection.onerror = function (error) {
            console.log('WebSocket Error ' + error);
            $scope.$apply(function () {
                $scope.errors.push(error);
            });
        };

        connection.onmessage = function(e) {
            $scope.$apply(function () {
                $scope.state = JSON.parse(e.data);
            });
        };
        $scope.connection = connection;
    };

    $(window).on("pageshow", function() {
        $scope.NewConnection();
    });

    $(window).on("pagehide", function() {
        $scope.connection.close();
    });

    $scope.addCollection = function(name) {
        $.ajax({
            url: "/collection/?" + $.now(),
            type: "POST",
            cache: false,
            data: {
                "name": name},
            statusCode: {
                404: function() {
                },
                200: function() {
                }
            },
            dataType: "html"
        });
        $scope.collectionName = "";
    };

    $scope.addURL = function(collectionName, url) {
        $.ajax({
            url: "/collection/" + collectionName + "/?" + $.now(),
            type: "POST",
            cache: false,
            data: {
                "url": url},
            statusCode: {
                404: function() {
                },
                200: function() {
                }
            },
            dataType: "html"
        });
        $scope.url = "";
    };

}

