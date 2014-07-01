Dpacman
=======

[![Build Status](https://drone.io/github.com/teambox/dpacman/status.png)](https://drone.io/github.com/teambox/dpacman/latest)

Package manager for Docker-based applications

Download
--------
```
$ wget https://drone.io/github.com/teambox/dpacman/files/dpacman.gz
$ gunzip dpacman.gz
```

Build
-----
```
$ go get
$ go build
```

Build a package
---------------
```
$ sudo ./dpacman build example/
$ ls /var/lib/pacman/successful/test-package-0.0.1-1714124087/
Dpacman  images  files  test-image-0.0.1-1.tar.gz
```

Install a package
---------------
```
$ sudo ./dpacman install /var/lib/pacman/successful/test-package-0.0.1-1714124087/test-image-0.0.1-1.tar.gz
Unpacking package...
2014/07/01 08:35:33 Running pre-install script...
2014/07/01 08:35:33 Importing images
2014/07/01 08:35:33 Imporing image busybox:latest...
2014/07/01 08:40:07 Installing files...
2014/07/01 08:40:07 Running post-install script...
2014/07/01 08:40:07 Creating installation mark...
2014/07/01 08:40:07 Cleaning package's tmp folder...
```
