+++
title = "April 25, 2022 at 3:06 PM"
date = "2022-04-25T15:06:07"
draft = false
[params]
  source = "linkedin"
  likes = 0
  views = 0
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:6924372981729161217"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:6924372981729161217/"
+++

Algorithms can have serious consequences on lives of people around you!

I've already posted about the scandal the Dutch tax office was involved.

They used the second nationality as a feature in their model — to find possible fraudulent behavior in their allowances scheme.

There were two problems with their approach:

- First of all, it was unlawful in the NL. This was the biggest issue, algorithm, or no algorithm
- The second one, was that the algorithm didn't say **why** it flagged an individual.

Is this problematic?

Yes, it is! If you don't know why someone if flagged, then you will be looking into *everything* trying to find something is wrong. And sometimes that something is a technicality such as forgetting to sign a form — a far cry from committing fraud!

So how do you do it right?

A couple of years ago, I was called by a bank that had a very high performing machine learning model (an isolation forest) to flag correspondent banking transactions that were suspicious.

The problem is that isolation forests are not very explainable, you don't know why they flag something.

However the bank found it unacceptable for the model to just report a transaction to an analyst.

The analyst would have engaged in the same behavior the Dutch office engaged it: find anything that was not 100% kosher. Of course if you're not 100% within the lines, it doesn’t mean you're committing fraud. It can be as silly as forgetting to sign a form.

What I did back then was to develop a geometric model that would explain why the isolation forest model was flagging transactions.

Please do the same with models that can have nefarious effects. I don't care if you're wrong about my taste in fashion when I browse Amazon.

I very much care if my life gets destroyed though!

#machinelearning #frauddetection #tax #explainableai
