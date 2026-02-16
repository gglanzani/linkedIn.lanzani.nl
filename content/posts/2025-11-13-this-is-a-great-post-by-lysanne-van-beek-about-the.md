+++
title = "November 13, 2025 at 10:46 AM"
date = "2025-11-13T10:46:21"
draft = false
[params]
  source = "linkedin"
  likes = 26
  views = 1267
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:7394687055584317440"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:7394687055584317440/"
  image_sources = ["https://media.licdn.com/dms/image/sync/v2/D4E27AQHDjlrVYHlHmw/articleshare-shrink_1280_800/B4EZp8zBbtGcAU-/0/1770335004813?e=2147483647&v=beta&t=eorRKwLPbTOYwk-I_fxkql3J9kjd68wD_twPP3J_ykQ"]
+++

This is a great post by Lysanne van Beek about the misconceptions and surprises she encountered during our Intro to AI course!

One of the surprises is that ChatGPT never gives the same answer twice, and Lysanne goes a bit into the topic, but it's actually much more fascinating!

LLMs have the concept of temperature, which basically says how creative the model should be with its responses. High temperature translates to higher creativity.

However, even if we set the temperature to 0—instructing the model to always pick the most probable word next—the model will give different answers to the same prompt.

Why?

The reason is… speed! While the details are complicated, the general idea is as follows: to provide an answer quickly, the computation is distributed across different machines. These machines can round floating-point numbers slightly differently because the values stored in memory are slightly different. Small differences can create significantly different probabilities, meaning… You never get the same answer!

I found this part really impactful, as even though you set the temperature to 0, models cannot offer reproducibility.

This makes having a solid evaluation framework even more important to ensure the model continues to behave in the desired way.

https://xebia.com/blog/7-things-that-always-surprise-people-in-our-intro-to-ai-course/

![Post image 2](/media/2025-11-13-this-is-a-great-post-by-lysanne-van-beek-about-the-image2.jpg)
