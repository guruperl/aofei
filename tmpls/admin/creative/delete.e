<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Creative Deleted.</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
  </head>
  <body>
    <script>      
      $( document ).ready(
        function() {
          var url = window.location + ""
          var idx = url.indexOf( "itemid=" ) 
          var pair = url.substring( idx, url.length )
          var itemid = pair.split( '=' )[ 1 ]
          var pageURL = "/go.fcgi/admin/e/creative?action=topics&itemid=" + itemid

          $( '#redirect' ).click(
            function() {
              window.location = pageURL
            }  
          )  
        }  
      )    
    </script>
    <div align="center">
      <label>Creative deleted.</label>
      <br/>
      <input id="redirect" name="redirect" type="button" value="Back to Admin Page" />
    </div>      
  </body>  
</html>
