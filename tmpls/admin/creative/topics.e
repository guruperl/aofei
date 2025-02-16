<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Creative Management</title>
    [% INCLUDE topics_ui.e %]
  </head>
  <body>
     
    <div align="center">
      <h2 align="center" class="curTitle">Creative Management</h2>
       <h2 align="center" class="curTitle"><a href='adv'>Adv Management</a> : <a href="campaign?action=topics&advid=[% advid %]&company_esc=[% company_esc %]">[% company%]</a> : <a href="item?action=topics&campaignid=[% campaignid %]&campaignname_esc=[% campaignname_esc %]&advid=[% advid %]&company_esc=[% company_esc %]">[% itemname %]</a> > Creative Management</h2>
      <div class="navContainer">
      <div class="navList">
        <a id="logout" href="../../../go.cgi?action=logout&role=admin">Log out</a>
      </div>
      <!--    
      <table>            
        <tr>
          <td>
            <input id="deleteSelection" type="button" value="Delete Selected Creatives" />
          </td>  
        </tr>
      </table>
      -->
      <table id="tblGrid" border="1" class="tblGrid">
        <thead>
          <tr class="title">     
            <th>Creative ID</th>
            <th>Creative Name</th>
            <th>Content</th>
            <th>Moment</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>  
        </thead>
        <tbody>
          [% FOREACH row IN topics %]
            <tr id="[% GET row.itemid %]_[% GET row.creativeid %]" class="Row">
              <td>
                <a href="creative?action=edit&creativeid=[% GET row.creativeid %]&itemid=[% GET row.itemid %]">[% GET row.creativeid %]</a>
              </td>
              <td>[% GET row.creativename %]</td>
              <td>[% GET row.content %]</td>
              <td>[% GET row.moment %]</td>
              <td style="text-align: center"><img src="/uilib/comImg/[% GET row.status %].png" /></td>
              <td>
                <a href="creative?action=edit&creativeid=[% GET row.creativeid %]&itemid=[% GET row.itemid %]"><img src="/uilib/comImg/editor.gif" border=0 alt="Edit" /></a>
                <a href="javascript:execConfirmHrefDelete('creative?action=delete&creativeid=[% GET row.creativeid %]&itemid=[% GET row.itemid %]','e')"><img src="/uilib/comImg/delete.gif" border=0 alt="Delete" /></a>
              </td>            
            </tr>
          [% END %]
        </tbody>
      </table>
    </div>  
  </body>
</html>
