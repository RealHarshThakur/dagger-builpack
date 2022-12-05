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


## Stack
- Why Buildkit?
Buildkit has been battle-tested by [Netflix](https://news.ycombinator.com/item?id=30858863) and dominates container build eco-system(directly/indirectly). It is possible to horizontally scale it as required while sharing cache.

- Why Buildpacks?
Buildpack is de-facto standard to go from code to image. It makes it easy to manage a fleet of application images. This project relies on buildpack spec alone and doesn't use Pack library/clients for builds as they're tied to Docker client as of today. 

- Why Dagger?
Before Dagger, way to interact with Buildkit was via buildkit frontend or by making API calls. Both of which require understanding of how Buildkit works. I looked at Dagger as a user-friendly wrapper over Buildkit API.

- Syft/Grype: They're just amazing tools at what they do. I didn't consider other ones but let me know if you would like to use other tools to generate SBOM/Vulnerabiltiy reports. 

- Why not Tekton?
Tekton shows an example of how to use Kaniko to build images. To share cache, it has a concept of workspace but automating it didn't seem trivial. I'm open to suggestions if you have any.

- Why not kpack?
kpack seems to have a concept of image caches but it's not as implict as Buildkit. Cache is a first-class citizen in Buildkit and it's easy to share it across multiple nodes. 

### Example:

Run the following command:
```
dagger-buildpack -g https://github.com/RealHarshThakur/sample-golang 
```

It will push the image to ttl.sh, produce sbom.json and vuln.json in `artifacts` directory

If you have a remote buildkit hosted, you can export `BUILDKIT_HOST"="tcp://<ip address>` for image builds to be done remotely.


### Notable links
- Buildpack spec: https://github.com/buildpacks/spec/blob/main/platform.md
- Can we use distroless builders?
https://github.com/buildpacks/pack/issues/42