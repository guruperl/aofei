[% INCLUDE start.e %]

<div class="ui-layout-west">
<ul id="treeList">
        <li><a href="campaign?action=edit&campaignid=[% campaignid %]&campaignname_esc=[% campaignname_esc %]">[% campaignname %]</a>
            <p></p>
            <ul>
            <li><a href="item?action=edit&itemid=[% insert.0.itemid %]&campaignid=[% campaignid %]&campaignmd5=[% campaignmd5 %]&campaignname_esc=[% campaignname_esc %]">[% itemname %]</a></li>
        </ul></li>
</ul>
</div>
<div class="ui-layout-center">

added.

</div>

[% INCLUDE end.e %]
