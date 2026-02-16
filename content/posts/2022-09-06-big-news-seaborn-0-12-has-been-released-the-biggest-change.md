+++
title = "September 6, 2022 at 8:35 PM"
date = "2022-09-06T20:35:23"
draft = false
[params]
  source = "linkedin"
  likes = 0
  views = 0
  comments = 0
  url = "https://www.linkedin.com/feed/update/urn:li:share:6973015819802796032"
  linkedin_analytics_url = "https://www.linkedin.com/analytics/post-summary/urn:li:activity:6973015819802796032/"
+++

Big news, seaborn 0.12 has been released.

The biggest change is the object API, inspired by the grammar of graphics.

But I can’t possibly be the only one bothered by one of their examples?

Especially the y0 and y1 feels super implicit and group="Country", col=None feels like coming out of an ancient book written on knowledge long forgotten.

Am I ruined by my years using matplotlib during my PhD?

so.Plot(healthexp, ×="Year" )
.pair(y=["Spending_USD", "Life_Expectancy"])
. facet(col="Country", order=country order)
•scale(x=so. Continuous(). tick(at=[1985, 2020]))
. label(x=""
y0= "Spending (USD)", y1="Life expectancy")
.add(so.Line(color="#ceced3" ), group="Country", col=None)
.add(so.Line(linewidth=3), color="Country", legend=False)

https://michaelwaskom.medium.com/announcing-the-release-of-seaborn-0-12-f26266ddbd8f
