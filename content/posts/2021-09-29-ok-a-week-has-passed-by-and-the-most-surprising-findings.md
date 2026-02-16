+++
title = "September 29, 2021 at 10:40 AM"
date = "2021-09-29T10:40:23"
draft = false
[params]
  source = "linkedin"
  likes = 1
  views = 0
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:6848929432040747008"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:6848929432040747008/"
  image_sources = ["https://media.licdn.com/dms/image/v2/C4D22AQHe48p9s_wyYg/feedshare-shrink_800/feedshare-shrink_800/0/1632337437684?e=2147483647&v=beta&t=8rAtATMnW5qJ-tbVarA24g1CWeBdN8zN2JzY_1x79sY"]
+++

OK, a week has passed by and the most surprising findings are (🥁🥁🥁):

- No access to camera, location, microphone, photos from unexpected apps. This is to be expected, iOS is pretty bad ass with keeping tabs on this
- The Economist and the Xiaomi home app are absolutely awful in the number of requests they make to various tracking domains
- Youtube — together with the Xiaomi home app, albeit less frequently — is the king of connecting to bare IP addresses, instead of using domain names. Why do they do it? Because it's impossible to block these at DNS level, you have to block at the firewall level. This is something not all "normal" people can do: you need special gear at home
- The identifier of Twitter for iPhone is still com.atebits.Tweetie2. Bonus points if you know to what it refers!
- Lots of apps use Firebase (https://firebase.google.com). No surprises, Firebase tagline is "Firebase helps you build 
and run successful apps"

What did I do with all this information?

Mostly, I blocked extra domains at the DNS level.

I also have a couple of apps where I might want to investigate further what they do and why, and maybe block a couple of IP addresses on a DNS level.

The results of course are not shocking as I try to vet most apps I install, I don't use Whatsapp, Instagram, Facebook, etc., if an app is ad-based I usually look into a payed alternative, etc.

I can only imagine what you get on platforms — ahem Android — where privacy is not a USP 🙈

![Post image 2](/media/2021-09-29-ok-a-week-has-passed-by-and-the-most-surprising-findings-image2.jpg)
