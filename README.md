# Mediasync Server

**WIP: This is a WIP, things will not work.**


**A simple client to copy files via HTTP.**

## What is Mediasync client

I have a machine outside of my local that automatically download files. Because I want these available to me on my local network, I rsync these periodically. The goal of this project is to download from this server instead, and use webhooks to make the syncs on-demand instead of at specific times. I also want to eliminate the need for ssh transport, and I want the server to be able to run in k8s. So I decided to create something for myself.
