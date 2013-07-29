module.exports = function(grunt) {

    // Project configuration.
    grunt.initConfig({
        pkg: grunt.file.readJSON('package.json'),
        banner: '/**\n' +
                '* <%= pkg.name %>.js v<%= pkg.version %> \n' +
                '* <%= grunt.template.today("yyyy") %> \n' +
                '*/\n',
        clean: {
            static: ['static']
        },
        concat: {
            options: {
                banner: '<%= banner %>',
                stripBanners: false
            },
            ginger: {
                src: ['bower_components/jquery/jquery.js', 'bower_components/angularjs/index.js', 'bower_components/angular-ui-bootstrap/index.js', 'js/ginger.js', 'bower_components/bootstrap/dist/js/bootstrap.js'],
                dest: 'static/js/<%= pkg.name %>.js'
            }
        },
        uglify: {
            options: {
                banner: '<%= banner %>'
            },
            ginger: {
                files: {
                    'static/js/<%= pkg.name %>.min.js': ['<%= concat.ginger.dest %>']
                }
            }
        },
        jshint: {
            options: {
                jshintrc: 'js/.jshintrc'
            },
            gruntfile: {
                src: 'Gruntfile.js'
            },
            src: {
                src: ['js/*.js']
            },
            test: {
                src: ['js/tests/unit/*.js']
            }
        },
        recess: {
            options: {
                compile: true
            },
            ginger: {
                files: {
                    'static/css/ginger.css': ['bower_components/bootstrap/less/bootstrap.less']
                }
            },
            min: {
                options: {
                    compress: true
                },
                files: {
                    'static/css/ginger.min.css': ['bower_components/bootstrap/less/bootstrap.less']
                }
            }
        },
    });

    // These plugins provide necessary tasks.
    grunt.loadNpmTasks('grunt-contrib-uglify');
    grunt.loadNpmTasks('grunt-contrib-jshint');
    grunt.loadNpmTasks('grunt-contrib-clean');
    grunt.loadNpmTasks('grunt-contrib-concat');
    grunt.loadNpmTasks('grunt-recess');

    // Test task.
    grunt.registerTask('test', ['jshint']);

    // JS distribution task.
    grunt.registerTask('static-js', ['concat', 'uglify']);

    // Default task(s).
    grunt.registerTask('default', ['uglify']);

    // CSS distribution task.
    grunt.registerTask('static-css', ['recess']);

    // Full distribution task.
    grunt.registerTask('static', ['clean', 'static-css', 'static-js']);

    // Default task.
    grunt.registerTask('default', ['test', 'static']);

};
