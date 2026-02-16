+++
title = "April 18, 2025 at 5:49 AM"
date = "2025-04-18T05:49:05"
draft = false
[params]
  source = "linkedin"
  likes = 0
  views = 6437
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:7318873185288871937"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:7318873185288871937/"
  image_sources = ["https://media.licdn.com/dms/image/v2/D4E22AQE1v502v7zQFQ/feedshare-shrink_800/B4EZZHcYQnHkAk-/0/1744955344460?e=2147483647&v=beta&t=mfeK0t_F-WJfC-GeE5t4ou4J-RqKev5Ih5utUyhdQKg"]
+++

I saw an interesting thing pop up lately—why copy pasting code into an LLM is better than using AI-assisted coding tools such as GitHub Copilot.

The reason boils down to token compression.

Token compression is the technique to limit the number of tokens sent to the LLM. When a code base gets large, token compression makes a huge difference in terms of speed and cost. If you're using a all-you-can-code (flat-fee) AI-assisted coding tool, it is in the best interest of the provider to compress your code as much as possible.

But what do you leave out? When applying token compression to text, the choice is easy: leave out the the's, and's, and so on. With code, the choice is more difficult.

And that's why copy pasting the whole code base to an LLM, can produce better results: the LLM will have all relevant tokens available, not just a subselection!

The quickest way to try it out is the the amazing files-to-prompt and llm libraries from Simon Willison. See the image for a quick start!

https://github.com/simonw/files-to-prompt
https://github.com/simonw/llm

![Post image 2](/media/2025-04-18-i-saw-an-interesting-thing-pop-up-lately-why-copy-pasting-image2.jpg)
