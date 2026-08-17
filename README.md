# ingress-nginx (Rubin Observatory DM-SQuaRE fork)

## USE AT YOUR OWN RISK

We're not ingress-nginx maintainers, and we are not qualified to be.
If you found this repository while you were looking for some way to put a more modern NGINX into ingress-nginx-controller, you're welcome to try.
However, it's entirely on you to succeed or fail, and we accept no responsibility if something horrible happens to your Kubernetes deployment if you use this package.
We offer no support, and issues filed against it will most likely be cheerfully ignored.

## Rationale

This repository is designed to be a stopgap until Rubin Observatory can migrate to the K8s Gateway API.

We accept that [ingress-nginx is no longer maintainable](README-orig.md).

Nevertheless, we were unable to move away from ingress-nginx by the maintenance deadline and then the subsequent discovery of a major vulnerability.
Thus we've decided to rebuild the ingress-nginx controller with a version of NGINX that does not include the vulnerability.

## Changes from the parent repository

In addition to the obvious changes to the nginx base image and what repository it lives in, we've done a few other things:

 * We dropped almost all of the GitHub Actions, keeping only one that rebuilds the NGINX base container and the controller container.  We are not intending to maintain this package as a going concern.
 * We updated the GitHub Actions to their current (June 5, 2026) major versions and let them float within that major version.
 * We dropped 32-bit ARM support, leaving only amd64 and arm64 architectures.  Nothing in the Rubin environment that runs Kubernetes will ever need 32-bit ARM.
 * We added instructions for updating to a new version of NGINX, which is the only maintenance action we ever intend to take.
 * We dropped the patch to nginx that had already been addressed upstream.
 * The `log_escape_non_ascii` patch needed rework to fit an updated source file.
 * Note that the controller name is now `ingress-nginx-controller`, not simply `controller`.  That's because `lsst-sqre` supplies other controllers (such as [Nublado](https://nublado.lsst.io)).  You will need to be aware of this when updating your helm charts.

## How to update NGINX (instructions for Rubin DM SQuaRE)

We're probably going to have to do this again before we've moved away from ingress-nginx.
Here's the process for updating the underlying NGINX version.

If you're doing this for some other project, you will need to read on past this section as well.

### Choose your version

Decide on a version.
Go to https://nginx.org/download and pick the version you want.
Once you've done that, you should get its sha256sum.
Do something like the following:

```bash
NGINX_VERSION=1.30.4
wget https://nginx.org/download/nginx-$NGINX_VERSION.tar.gz
sha256sum nginx-$NGINX_VERSION.tar.gz | awk '{print $1}'
```

### Get your patch set working

#### Set up an environment

You probably want to do this on Linux rather than MacOS, or, God forbid, Windows.
An EC2 instance is a quick and easy way to get a very basic Linux machine going.
Note that `quilt` is not available for EC2 Linux; I used a Debian image instead.

#### Set up quilt, sources, and patches

First, acquire quilt; it is very probably in whatever package manager you're using.

Second, grab a fresh copy of NGINX from [https://nginx.org/downloads](https://nginx.org/downloads).

Unpack the vanilla NGINX sources.

Copy the `patches` subdirectory from [images/nginx/rootfs/patches](images/nginx/rootfs/patches) into the top-level NGINX source directory.

#### Try patching, in dry-run mode

Change directory into the top-level NGINX source directory.
Try applying patches:

```bash
for p in patches/*; do echo "*** ${p} ***"; patch --dry-run -p1 < ${p}; done
```

Ignore anything that patches with a fuzz offset; that is fine and we will get to it in the next step.
What you are concerned with are patches that are rejected.

For each of these, see what went wrong.
The best case is that it's a patch that has already been incorporated upstream, in which case you can just delete it.
Otherwise, you're going to have to put in some work to determine why it failed and how to make it work.

A common case is that a later patch depends on an earlier patch; in that case you may need to make a working copy of the repository, and actually apply the patches in sequence to be able to tell what works and what does not.

Eventually, you will either have removed all patches that failed to apply, or gotten them to apply, possibly with fuzz.

Now it's time to rebase the patch set to eliminate the fuzz.

#### Rebase the patch set

Create a series file from the existing patches: `cd patches && quilt import *`.

Now do the rebase:

```bash
quilt pop -a
while quilt push; do quilt refresh -p ab; done
```

This will create (assuming that the patches all applied, which they should have if you did the iterative process above) modified patch files, and backup patch files with `~` extensions.


#### Tidy up

Remove all your backup files and the quilt series file: `rm series *~`

Rename the patches so they have the current NGINX version.
For instance if you are moving from version 1.30.2 to 1.30.4, do:
```bash
OLD=1.30.2
NEW=1.30.4
for p in $(ls *-${OLD}-*.patch); do n=$(echo $p | sed -e "s/${OLD}/${NEW}/"); mv ${p} ${n}; done
```

#### Test patch application

Make sure all patches apply cleanly.
Unpack the vanilla NGINX tarball somewhere new, cd to it, and copy your patch directory over to it.
Then apply all the patches.
```bash
mkdir path-to-test-nginx
cd path-to-test-nginx
tar xvpfz path-to-tarball.tgz
cd nginx-${NEW}
cp -a path-to-patching-nginx/patches .
for p in patches/*.patch; do patch -p1 < ${p}; done
```

This should apply cleanly.  If it does, move on to updating the patches in the ingress-nginx git repository.  If not, work on the patch set until it does.

#### Update patches in ingress-nginx

Make a new branch of this repository (if you're part of DM SQuaRE, presumably `tickets-DM/something` or `t/DM-something`).

Go back to [images/nginx/rootfs](images/nginx/rootfs) and remove the extant patchset: `git rm -rf patches`.

Copy the rebased patches into place and add them to git: `cp -a path-to-nginx-src/patches . && git add patches`.

Commit the changes: `git commit -m "Rebase patch set"`

Now it's time to fix up the rest of the build process.

### Update files

Edit [images/nginx/rootfs/build.sh](images/nginx/rootfs/build.sh).
Change `NGINX_VERSION` on line 21 to the version you selected.
Then change the checksum for the NGINX package on line 192 (which starts with `get_src`) to the checksum you just extracted.

Increment the version numbers of the containers.
[TAG](TAG) holds the controller tag, and [images/nginx/TAG](images/nginx/TAG) holds the NGINX base container tag.
Keep the `-devsquare` label that's appended to the tag.

Change [NGINX_BASE](NGINX_BASE) to pull the base image with the new base container tag.

Update [Changelog.md](Changelog.md) to note the NGINX upgrade.

Commit and push your changes, then open a pull request.

### Let the build run

GitHub Actions will do the base container build and the controller build.
It takes about two hours to rebuild the base container.
The controller is very quick after the base container is done.

### After the build

When everything is finished, `ghcr.io/lsst-sqre/nginx:NGINX_TAG` and `ghcr.io/lsst-sqre/ingress-nginx-controller:CONTROLLER_TAG` should both exist, and you can update [Phalanx](https://phalanx.lsst.io) to use the new controller container image.

### Git tidying

Merge your PR, and then create and push a git tag with the new value in [TAG](tag).

## How to update if you're not Rubin DM SQuaRE

You have to do all the steps above, but also you're going to need to change `ghcr.io/lsst-sqre` to whatever your container registry is.

[.github/workflows/build.yaml](.github/workflows/build.yaml) contains two instances of the `REGISTRY` env key with that value.  Also, if you're not using `ghcr.io`, you'll need to change the authentication information in three places, one for each `Login to GitHub Container Registry` step.

Then you'll need to change the `REGISTRY` definitions in [images/nginx/Makefile](images/nginx/Makefile) and [Makefile](Makefile).

The tags in [TAG](TAG), [images/nginx/TAG](images/nginx/TAG), and [NGINX_BASE](NGINX_BASE) should also probably get a label that is something other than `-devsquare`; likewise for your [Changelog.md](Changelog.md) entry.

## Conclusion

This kicks the can down the road a little farther.
It's currently August 13, 2026.
Let's see how long it takes us to move away from ingress-nginx entirely.
