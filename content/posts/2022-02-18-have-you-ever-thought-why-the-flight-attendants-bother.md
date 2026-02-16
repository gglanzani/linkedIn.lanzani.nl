+++
title = "February 18, 2022 at 8:59 PM"
date = "2022-02-18T20:59:44"
draft = false
[params]
  source = "linkedin"
  likes = 18
  views = 0
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:6900544374120292352"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:6900544374120292352/"
  image_sources = ["https://media.licdn.com/dms/image/v2/C5622AQFh84UFvCn0mA/feedshare-shrink_800/feedshare-shrink_800/0/1645217983972?e=2147483647&v=beta&t=E9ttvWMnFCwO94qF6I2vW4VxbEcFNc216SKfhdZSslQ"]
+++

Have you ever thought why the flight attendants bother giving safety instructions? Do you listen to them?

Flight attendants are stuck. They can’t go off script.

Probably a long time ago, there were tests on how to deliver those safety instructions to passengers.

The current way was tested not with busy passengers needing to get somewhere, but people recruited for the purpose. It probably fared better than anything else.

Yet, when applied in real life, it sucks. We don’t listen to what they say.

I see the same mistake made in data science: people test their model with real data, but not in production.

I used to tell my classes a story of a big online retailer developing a much better version of their recommender — “customers who bought this, also bought that” type of thing.

With the new recommender, fewer clicks were necessary to understand the set of items the customer wanted to buy.

Before rolling out, they A/B tested it — luckily.

To their surprise, people exposed to the new version, were closing their browser more quickly **without** buying!

Some of them were logged in, so they decided to investigate.

It turns out, customers were creeped out by the eerie accuracy of the new recommender. They left the website, afraid of what else the retailer would find out about them.

The retailer went back to the old version.

It doesn’t matter how enthusiast data scientists are about the model. 

Without testing in production, it counts for nothing.

![Post image 2](/media/2022-02-18-have-you-ever-thought-why-the-flight-attendants-bother-image2.jpg)
