(provide "go" "1.16")
(env/set "GO111MODULE" "on")
(env/set "GOSUMDB" "off")

(env/set "GOROOT" (path-concat (env/get "HOME") ".dev" "go" "1.16"))
(env/prepend-to-pathlist "PATH" (path-concat (env/get "GOROOT") "bin"))

(env/append-to-pathlist "GOPATH" (path-concat (env/get "HOME") "go"))
(env/append-to-pathlist "PATH" (path-concat (env/get "GOPATH") "bin"))

(env/append-to-pathlist "GOPATH" (env/get "PKG_PATH"))
(env/append-to-pathlist "PATH" (path-concat (env/get "GOPATH") "bin"))
