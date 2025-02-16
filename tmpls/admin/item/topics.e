<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Item Management</title>
    <script type="text/javascript" src="/js/jquery-1.4.2.min.js"></script>
[% INCLUDE topics_ui.e %]
  </head>
  <body>
   
    <div align="center">  
       <h2 align="center" class="curTitle"><a href='adv'>Adv Management</a>: <a href="campaign?action=topics&advid=[% advid %]&company_esc=[% company_esc %]">[% campaignname %]</a> > Item Management</h2>
 <div class="navContainer">
      <div class="navList">
        <a id="logout" href="../../../go.cgi?action=logout&role=admin">Log out</a>
      </div>
</div> 
<!--     
      <table>
        <tr>
          <td>
            <input id="btnColumns" type="button" />
          </td>          
          <td>
            <input id="btnDeleteSelection" type="button" value="Delete Selected Campaigns" />
          </td>
        </tr>
      </table>
      -->
      <br/>      
      <table id="tblGrid" border="1" class="tblGrid">
        <thead>
          <tr>
            <th>Item&nbsp;ID</th>
            <th>Item&nbsp;Name</th>
            <th>Size&nbsp;ID</th>
            <th>Start</th>
            <th>End</th>
            <th>Quantity</th>
            <th>Cost</th>
            <th>Cost&nbsp;Type</th>            
            <th>Delivery</th>
            <th class="Invisible">CPM&nbsp;FC</th>
            <th>Status</th>
            <th>Operator</th>
          </tr>  
        </thead>
        <tbody>
          [% FOREACH row IN topics %]
            <tr id="[% GET row.campaignid %]_[% GET row.itemid %]" class="Row">
              <td style="text-align: right">
              <a href="creative?action=topics&itemid=[% GET row.itemid %]&itemname_esc=[% GET row.itemname %]&campaignid=[% GET row.campaignid %]&campaignname_esc=[% campaignname_esc %]&advid=[% advid %]&company_esc=[% company_esc %]">[% GET row.itemid %]</a>
              <!--<a href="item?action=edit&itemid=[% GET row.itemid %]&campaignid=[% GET row.campaignid %]">[% GET row.itemid %]</a>-->
              </td>
              <td nowrap>[% GET row.itemname %]</td>              
        
              <td style="text-align: right">[% GET row.sizeid %]</td>    
              <td nowrap>[% GET row.startx %]</td>
              <td nowrap>[% GET row.endx %]</td>
              <td style="text-align: right">[% GET row.quantity %]</td>              
              <td style="text-align: right">[% GET row.cost %]</td>
              <td style="text-align: right">[% GET row.costtype %]</td>
              <td>[% GET row.deliveryrate %]</td>
              <td class="Invisible" style="text-align: right">[% GET row.cpm_fc %]</td>
              <td style="text-align: center"><img src='/uilib/comImg/[% row.status %].png' /></td>            
              <td>
                <a href="item?action=edit&itemid=[% GET row.itemid %]&campaignid=[% GET row.campaignid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>
                &nbsp;&nbsp;
                <a href="javascript:execConfirmHrefDelete('item?action=delete&itemid=[% GET row.itemid %]&campaignid=[% GET row.campaignid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete Item" /></a>
              </td>              
            </tr>
          [% END %]
        </tbody>
      </table>  
    </div>  
  </body>
</html>
