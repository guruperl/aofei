[% INCLUDE start.e %]

<div class="ui-layout-west">
<ul id="treeList">
        <li><a href="campaign?action=edit&campaignid=[% campaignid %]">[% campaignname %]</a>
			<p></p>
[% IF entitytype=='item' %]<ul><li><a href="item?action=edit&itemid=[% itemid %]&campaignid=[% campaignid %]&campaignmd5=[% campaignmd5 %]&campaignname_esc=[% campaignname_esc %]">[% itemname %]</a>
            <p></p>[% END %]
            <ul>
            <li>Targeting</li>
			</ul>
[% IF entitytype=='item' %]</li></ul>[% END %]
	</li>
</ul>
</div>

<link type="text/css" rel="stylesheet" media="screen" href="/js/list/jquery.toChecklist.min.css" />
<script type="text/javascript" src="/js/list/jquery.toChecklist.min.js"></script>
<script type="text/javascript" src="/js/demography.js"></script>
<script type="text/javascript">
[% IF age %]
var age1 = [% age.0.3 %] & 127;
var age2 = [% age.0.3 %] >> 7;[% ELSE %]
var age1 = 1;
var age2 = 128;[% END %][% IF hsize %]
var hsize1 = [% hsize.0.3 %] & 15;
var hsize2 = [% hsize.0.3 %] >> 4;[% ELSE %]
var hsize1=1;
var hsize2=16;[% END %][% IF income  %]
var income1 = [% income.0.3 %] & 15;
var income2 = [% income.0.3 %] >> 4;[% ELSE %]
var income1=1;
var income2=16;[% END %]
var incomes = [[  0,  9], [ 10, 19], [ 20, 29], [ 30, 39], [ 40, 49],
	[ 50, 59], [ 60, 69], [ 70, 79], [ 80, 99], [100,119], [120,149], 
	[150,199], [200,299], [300,499], [500,999], [999,999]]; 

	$('#countryselect').toChecklist({showSelectedItems : true});
	$('#stateselect').toChecklist({showSelectedItems : true});
	$('#cityselect').toChecklist({showSelectedItems : true});
	$('#dmaselect').toChecklist({showSelectedItems : true});
	$('#osselect').toChecklist({showSelectedItems : true});
	$('#browserselect').toChecklist({showSelectedItems : true});
	$('#languageselect').toChecklist({showSelectedItems : true});
	$('#bandwidthselect').toChecklist({showSelectedItems : true});

$(function() {
	$('select').toChecklist({showSelectedItems : true});
	$(".ui-layout-center").tabs();
	slider_get("age",1,128,age1,age2);
	slider_get("hsize",1,16,hsize1,hsize2);
	slider_get_ref("income",1,16,income1,income2,incomes);
});

$(document).ready(function() {
[% IF gender.defined %]checkbox_get("gender", [% gender.0.3 %]);[% END %]
[% IF marriage.defined %]checkbox_get("marriage", [% marriage.0.3 %]);[% END %]
[% IF children.defined %]checkbox_get("children", [% children.0.3 %]);[% END %]
[% IF ethnicity.defined %]checkbox_get("ethnicity", [% ethnicity.0.3 %]);[% END %]
[% IF education.defined %]checkbox_get("education", [% education.0.3 %]);[% END %]
[% IF religion.defined %]checkbox_get("religion", [% religion.0.3 %]);[% END %]
[% IF occupation.defined %]checkbox_get("occupation", [% occupation.0.3 %]);[% END %]
[% IF timetype.defined %]radio_get("timetype", [% timetype.0.3 %]);[% END %]
[% IF wkday.defined %]checkbox_get("wkday", [% wkday.0.3 %]);[% END %]
[% IF fullday.defined %]datepicker_get([% fullday.0.3 %], "date1", "date2");[% END %]
[% IF fullhour.defined %]time_get([% fullhour.0.3 %], "time1","time2");[% END %]
[% IF wkhour.defined %]time_get([% wkhour.0.3 %], "wktime1","wktime2");[% END %]

	$("#date1").datepicker({dateFormat: 'yy-mm-dd', showOn: 'button',
		buttonImageOnly: true, buttonImage: '/images/icon_cal.png'});
	$("#date2").datepicker({dateFormat: 'yy-mm-dd', showOn: 'button',
		buttonImageOnly: true, buttonImage: '/images/icon_cal.png'});
	$("#datesubmit").click(function(){datepicker_set("fullday", "date1", "date2")});

	$("#countrysubmit").click(function(){select_set("country")});
	$("#statesubmit").click(function(){select_set("state")});
	$("#citysubmit").click(function(){select_set("city")});
	$("#dmasubmit").click(function(){select_set("dma")});
	$("#ossubmit").click(function(){select_set("os")});
	$("#browsersubmit").click(function(){select_set("browser")});
	$("#blanguagesubmit").click(function(){select_set("blanguage")});
	$("#bandwidthsubmit").click(function(){select_set("bandwidth")});

	$("#zipsubmit").click(function(){simple_set("zip","valueid")});
	$("#zipdelete").click(function(){simple_delete("zip")});
	$("#areacodesubmit").click(function(){simple_set("areacode","valueid")});
	$("#areacodedelete").click(function(){simple_delete("areacode")});
	$("#companysubmit").click(function(){simple_set("company","name")});
	$("#companydelete").click(function(){simple_delete("company")});
	$("#ispsubmit").click(function(){simple_set("isp","name")});
	$("#ispdelete").click(function(){simple_delete("isp")});
	$("#contextsubmit").click(function(){simple_set("context","name")});
	$("#contextdelete").click(function(){simple_delete("context")});

	$("#gendersubmit").click(function(){checkbox_set("gender")});
	$("#marriagesubmit").click(function(){checkbox_set("marriage")});
	$("#childrensubmit").click(function(){checkbox_set("children")});
	$("#ethnicitysubmit").click(function(){checkbox_set("ethnicity")});
	$("#educationsubmit").click(function(){checkbox_set("education")});
	$("#religionsubmit").click(function(){checkbox_set("religion")});
	$("#occupationsubmit").click(function(){checkbox_set("occupation")});
	$("#timetypesubmit").click(function(){radio_set("timetype")});
	$("#wkdaysubmit").click(function(){checkbox_set("wkday")});
	$("#timesubmit").click(function(){time_set("fullhour", "time1", "time2")});
	$("#wktimesubmit").click(function(){time_set("wkhour", "wktime1", "wktime2")});
    $("#agesubmit").click(function(){slider_set('age',7)});
	$("#incomesubmit").click(function(){slider_set('income',4)});
	$("#hsizesubmit").click(function(){slider_set('hsize',4)});
});

</script>

<div class="ui-layout-center">
	<ul>
		<li style="font-size: 14px;"><a href="#tabs-1">Country</a></li>
		<li style="font-size: 14px;"><a href="#tabs-2">Geographic</a></li>
		<li style="font-size: 14px;"><a href="#tabs-3">Browser & OS</a></li>
		<li style="font-size: 14px;"><a href="#tabs-4">Context</a></li>
		<li style="font-size: 14px;"><a href="#tabs-5">Demographic</a></li>
		<li style="font-size: 14px;"><a href="#tabs-6">Custom</a></li>
		<li style="font-size: 14px;"><a href="#tabs-7">Date Time</a></li>
	</ul>

	<div class="ui-layout-content">
		<div id="tabs-1">[% INCLUDE tab1 %]</div>
		<div id="tabs-2">[% INCLUDE tab2 %]</div>
		<div id="tabs-3">[% INCLUDE tab3 %]</div>
		<div id="tabs-4">[% INCLUDE tab4 %]</div>
		<div id="tabs-5">[% INCLUDE tab5 %]</div>
		<div id="tabs-6">[% INCLUDE tab6 %]</div>
		<div id="tabs-7">[% INCLUDE tab7 %]</div>
	</div>

</div>

[% INCLUDE end.e %]
