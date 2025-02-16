<tr><td colspan=2> <h3>Sites to Target</h3> </td></tr>
<tr><td>Brand Awareness:</td><td>
<input type=radio name=brandweb [% IF item.brandweb=="0" %]checked[% END %] value=0>Unknown
<input type=radio name=brandweb [% IF item.brandweb=="1" %]checked[% END %] value=1>Sometimes
<input type=radio name=brandweb [% IF item.brandweb=="2" %]checked[% END %] value=2>Well-known
<input type=radio name=brandweb [% IF item.brandweb=="3" %]checked[% END %] value=3>Very well-known
</td></tr>
<tr><td>In Real World:</td><td>
<input type=radio name=brandreal [% IF item.brandreal=="0" %]checked[% END %] value=0>Unknown
<input type=radio name=brandreal [% IF item.brandreal=="1" %]checked[% END %] value=1>Sometimes
<input type=radio name=brandreal [% IF item.brandreal=="2" %]checked[% END %] value=2>Well-known
<input type=radio name=brandreal [% IF item.brandreal=="3" %]checked[% END %] value=3>Very well-known
</td></tr>
<tr><td>As Local Business:</td><td>
<input type=radio name=localbiz [% IF item.localbiz=="0" %]checked[% END %] value=0 checked>Unknown
<input type=radio name=localbiz [% IF item.localbiz=="1" %]checked[% END %] value=1>Sometimes
<input type=radio name=localbiz [% IF item.localbiz=="2" %]checked[% END %] value=2>Well-known
<input type=radio name=localbiz [% IF item.localbiz=="3" %]checked[% END %] value=3>Very well-known
</td></tr>
<tr><td>Domain Name:</td><td>
<input type=radio name=domainname [% IF item.domainname=="0" %]checked[% END %] value=0>Secondary or No Normal Domain
<input type=radio name=domainname [% IF item.domainname=="1" %]checked[% END %] value=1>Secondary Good 
<input type=radio name=domainname [% IF item.domainname=="2" %]checked[% END %] value=2>Normal
<input type=radio name=domainname [% IF item.domainname=="3" %]checked[% END %] value=3>Good or Short Name
</td></tr>
<tr><td>Domain Age:</td><td>
<input type=radio [% IF item.age=="0" %]checked[% END %] name=age value=0>New
<input type=radio [% IF item.age=="1" %]checked[% END %] name=age value=1>Since 2010
<input type=radio [% IF item.age=="2" %]checked[% END %] name=age value=2>Since 2006
<input type=radio [% IF item.age=="3" %]checked[% END %] name=age value=3>Since 2002
<input type=radio [% IF item.age=="4" %]checked[% END %] name=age value=4>Since 1998
</td></tr>
<tr><td>Site Visual:</td><td>
<input type=radio [% IF item.visual=="0" %]checked[% END %] name=visual value=0>Very Ugly
<input type=radio [% IF item.visual=="1" %]checked[% END %] name=visual value=1>Messy or Poor Looking
<input type=radio [% IF item.visual=="2" %]checked[% END %] name=visual value=2>Normal
<input type=radio [% IF item.visual=="3" %]checked[% END %] name=visual value=3>Good and Clean
</td></tr>
<tr><td>Window Popup:</td><td>
<input type=radio [% IF item.popup=="0" %]checked[% END %] name=popup value=0>2 Extra Windows or 3 JS Popups
<input type=radio [% IF item.popup=="1" %]checked[% END %] name=popup value=1>1 Window popup
<input type=radio [% IF item.popup=="2" %]checked[% END %] name=popup value=2>1 Javascript Popup
<input type=radio [% IF item.popup=="3" %]checked[% END %] name=popup value=3>No Popup
</td></tr>
<tr><td>Ad Density:</td><td>
<input type=radio [% IF item.crowd==0 %]checked[% END %] name=crowd value=0>Very Crow (8 or more ads)
<input type=radio [% IF item.crowd==1 %]checked[% END %] name=crowd value=1>Crowd (5-7 ads)
<input type=radio [% IF item.crowd==2 %]checked[% END %] name=crowd value=2>Normal (3-4 ads)
<input type=radio [% IF item.crowd==3 %]checked[% END %] name=crowd value=3>New (1-2 ads)
</td></tr>
<tr><td>UGC Ranking:</td><td>
<input type=radio [% IF item.ugc==0 %]checked[% END %] name=ugc value=0>X
<input type=radio [% IF item.ugc==1 %]checked[% END %] name=ugc value=1>NC-17
<input type=radio [% IF item.ugc==2 %]checked[% END %] name=ugc value=2>R
<input type=radio [% IF item.ugc==3 %]checked[% END %] name=ugc value=3>PG-13
<input type=radio [% IF item.ugc==4 %]checked[% END %] name=ugc value=4>PG
<input type=radio [% IF item.ugc==5 %]checked[% END %] name=ugc value=5>G
</td></tr>
<tr><td>Google Pagerank:</td><td>
<input type=radio [% IF item.pagerank==0 %]checked[% END %] name=pagerank value=0>&lt; 1
<input type=radio [% IF item.pagerank==1 %]checked[% END %] name=pagerank value=1>&lt; 2
<input type=radio [% IF item.pagerank==2 %]checked[% END %] name=pagerank value=2>&lt; 3
<input type=radio [% IF item.pagerank==3 %]checked[% END %] name=pagerank value=3>&lt; 4
<input type=radio [% IF item.pagerank==4 %]checked[% END %] name=pagerank value=4>&lt; 5
<input type=radio [% IF item.pagerank==5 %]checked[% END %] name=pagerank value=5>&lt; 6
<input type=radio [% IF item.pagerank==6 %]checked[% END %] name=pagerank value=6>&lt; 7
<input type=radio [% IF item.pagerank==7 %]checked[% END %] name=pagerank value=7>&lt; 8
<input type=radio [% IF item.pagerank==8 %]checked[% END %] name=pagerank value=8>&lt; 9
<input type=radio [% IF item.pagerank==9 %]checked[% END %] name=pagerank value=9>&le; 10
</td></tr>
<tr><td>Traffic Rank:</td><td>
<input type=radio [% IF item.traffic==0 %]checked[% END %] name=traffic value=0>Unavailable or Over 100k
<input type=radio [% IF item.traffic==1 %]checked[% END %] name=traffic value=1>Top 100k
<input type=radio [% IF item.traffic==2 %]checked[% END %] name=traffic value=2>Top 10k
<input type=radio [% IF item.traffic==3 %]checked[% END %] name=traffic value=3>Top 5000
<input type=radio [% IF item.traffic==4 %]checked[% END %] name=traffic value=4>Top 1000
<input type=radio [% IF item.traffic==5 %]checked[% END %] name=traffic value=5>Top 500
<input type=radio [% IF item.traffic==6 %]checked[% END %] name=traffic value=6>Top 100
<input type=radio [% IF item.traffic==7 %]checked[% END %] name=traffic value=7>Top 10
</td></tr>
<tr><td>Traffic Source:</td><td>
<input type=radio [% IF item.source==0 %]checked[% END %] name=source value=0>Spiderware
<input type=radio [% IF item.source==1 %]checked[% END %] name=source value=1>Domainer
<input type=radio [% IF item.source==2 %]checked[% END %] name=source value=2>Search or Index Optimized
<input type=radio [% IF item.source==3 %]checked[% END %] name=source value=3>Paid, or Emailed
<input type=radio [% IF item.source==4 %]checked[% END %] name=source value=4>Browser Bar
<input type=radio [% IF item.source==5 %]checked[% END %] name=source value=5>Proxy
<input type=radio [% IF item.source==6 %]checked[% END %] name=source value=6>Plugin
<input type=radio [% IF item.source==7 %]checked[% END %] name=source value=7>Normal
</td></tr>
<tr><td>Content Control:</td><td>
<input type=radio [% IF item.control==0 %]checked[% END %] name=control value=0>Blind copies 
<input type=radio [% IF item.control==1 %]checked[% END %] name=control value=1>User Provided (upload, chat or forum)
<input type=radio [% IF item.control==2 %]checked[% END %] name=control value=2>Copied Page or White Papers etc.
<input type=radio [% IF item.control==3 %]checked[% END %] name=control value=3>Controlled
</td></tr>
