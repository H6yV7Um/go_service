/**
 * Title:CKEditor插件示范
 * Author:铁木箱子(http://www.mzone.cc)
 * Date:2010-08-02
 */
CKEDITOR.plugins.add('thumb', {
    lang:['zh-cn','en'],
    requires: ['dialog'],
    init: function(editor){
        /*var b = a.addCommand('thumb', new CKEDITOR.dialogCommand('thumb'));
        a.ui.addButton('thumb', {
            label: a.lang.tbTip,
            command: 'thumb',
            icon: this.path + 'images/ck_thumb.png'
        });
        CKEDITOR.dialog.add('thumb', this.path + 'dialogs/thumb.js');*/
        editor.addCommand('thumb', new CKEDITOR.command(editor, {
            exec : function(editor){
                var e = editor.getSelection().getSelectedElement();
                if(e.getName() == 'img') {
                    var url = e.getAttribute('src');
                    var pos = url.lastIndexOf("/");
                    var filedir = url.substring(0, pos)
                    var filename = url.substring(pos, url.length)
                    $('#Thumb').val(filedir + "/thumb" + filename);
                }
            }
        }));
        editor.addCommand('upload_image', new CKEDITOR.command(editor, {
            exec : function(editor){
                var e = editor.getSelection().getSelectedElement();
                if(e.getName() == 'img') {
                    var url = e.getAttribute('src');
                }
                $.ajax({
                    type: "POST",
                    url: "/admin/upload_by_url",
                    data: {url: url},
                    success: function (data) {
                        if(data.Status != 0) {
                            alert(data.Error);
                        } else {
                            e.setAttribute('src', data.Url);
                            e.setAttribute('data-cke-saved-src', data.Url);
                            //$('#Thumb').val(data.Thumb);
                        }
                    }
                });
            }
        }));

        if(editor.addMenuItems){  //添加menu子项
            console.log("addMenuItems");
            editor.addMenuGroup('piggy', 3);
            editor.addMenuItems(  //have to add menu item first
                {
                    /*thumb:  //name of the menu item
                    {
                        label: '管理',
                        group: 'piggy',  //通过group可以设定本菜单项所属的组，多个组结合可以实现多层菜单。虽然这里不需要多层菜单，但是group必须定义
                        order: 3,
                        getItems : function() {
                            return {
                                thumb_setThumb : CKEDITOR.TRISTATE_OFF
                            };
                        }
                    },*/
                    thumb:
                    {
                        label : '设置为缩略图',
                        group : 'piggy',
                        command : 'thumb',
                        order : 22
                    },
                    upload_image:
                    {
                        label : '图片上传并替换',
                        group : 'piggy',
                        command : 'upload_image',
                        order : 24
                    }
                });
        }

        if (!editor.contextMenu)
            return;

        editor.contextMenu.addListener(function(element) {
            console.log("element=", element.getName());
            if(element.getName() == "img") {
                return {
                    thumb: CKEDITOR.TRISTATE_OFF,
                    upload_image: CKEDITOR.TRISTATE_OFF
                };
            }
            return null;
        } );
    }
});

