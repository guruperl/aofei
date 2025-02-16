[% INCLUDE start.e %]

[% SET row = xyz.0 %]
  <tr id="[% GET row.advid %]" class="Campaign">
    <td>
      <input id="[% GET row.campaignid %]" name="[% GET row.campaignid %]" class="cb" type="checkbox" />
    </td>
    <td>
      <div>[% GET row.campaignid %]</div>
      <input id="campaignid" name="campaignid" type="hidden" value="[% GET row.campaignid %]" />
    </td>
<!--    
    <td>
      <div id="campaignname" name="campaignname">[% GET row.campaignname %]</div>
    </td>
    <td>
      <div id="advertiser" name="advertiser">[% GET row.advid %]</div>
    </td>
    <td class="rowData">
      <a href="/go.fcgi/adv/e/campaign?action=edit_adv&advid=[% GET row.advid %]">Edit</a>
    </td>
    <td class="rowData">
      <a href="pub?action=deletecampaign&campaignid=[% GET row.campaignid %]">Delete</a>
    </td>    
-->    
  </tr>  

[% INCLUDE end.e %]
