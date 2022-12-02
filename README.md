## Dagger buildpack pipeline

Dagger-buildpack utilises couple open-source projects to build image from source code, generate SBOM(Software Bill of Materials) and scan the image for vulnerabilities.

You may find this useful if:
* You want to build container images from source code
* You don't want to rely on Docker 
* You don't want to create a Buildkit frontend
* You want to provide a remote container build service

## Open source projects it relies on:
* [Dagger](https://github.com/dagger/dagger)
* [Buildpacks](https://buildpacks.io/)
* [Buildkit](https://github.com/moby/buildkit)
* [Syft](https://github.com/anchore/syft)
* [Grype](https://github.com/anchore/grype)



### Example:

Run the following command:
```
dagger-buildpack -g https://github.com/RealHarshThakur/sample-golang 
```

If you have a remote buildkit hosted, you can export `BUILDKIT_HOST"="tcp://<ip address>` for image builds to be done remotely.


### Notable links
- Buildpack spec: https://github.com/buildpacks/spec/blob/main/platform.md
- Can we use distroless builders?
https://github.com/buildpacks/pack/issues/42