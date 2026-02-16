+++
title = "August 31, 2021 at 5:43 AM"
date = "2021-08-31T05:43:50"
draft = false
[params]
  source = "linkedin"
  likes = 0
  views = 0
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:ugcPost:6838345552481062912"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:6838345552481062912/"
+++

Where did Roman learn to optimize Spark?

Story time and bonus points if you guess the company's name based on the story!

While working there, one of us started using a Google library to interact with GCP. This library — unknown to us — had a bug that would cause listing the content of a bucket recursively.

Listing the content of a bucket is in principle not a big deal, the cost is minimal, but at the scale of this company, the cost quickly added up.

How much?

Well, after pushing to production the code with this library, we went to get lunch.

After coming back (a good hour later as this company — hint — has  terrific lunch facilities) the API calls looked completely off the charts: we raked up $50.000 in charges because of this bug during LUNCH (remember: implement monitoring! Had we not had it, finding out end of month would have meant MILLIONS of $ in charges).

We quickly rolled back the changes, looked why that was happening, fixed the changes, filed a PR to the Google library, and called it just a normal day. (Kudos to Google BTW to compensate us for the $50.000).

So what company are we talking about here?

And — more importantly — what kind of scale do they have if a tiny bug listing the content of a bucket costs you $50.000 dollars during lunch?

Well, this is the company Roman learned from. So you'd be a fool not to be at his event (and it's free!)
