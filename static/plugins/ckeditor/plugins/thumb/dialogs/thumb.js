/**
 * Title:CKEditor在线编辑器的代码插入插件
 * Author:Alex Xiang
 * Date:2016-05-03
 */
CKEDITOR.dialog.add('thumb', function(editor) {
	return {
		title: "缩略图", //editor.lang.dlgTitle,
　　 	minWidth: 360,
　　 	minHeight: 150,
　　 	contents: [{
　　 		id: 'cb',
　　 		name: 'cb',
　　 		label: 'cb',
　　 		title: 'cb',
　　 		elements: [{
　　 			type: 'textarea',
　　 			required: true,
　　 			label: "test", // editor.lang.mytxt,
　　 			style: 'width:350px;height:100px',
				rows: 6,
　　 			id: 'mytxt',
　　 			'default': 'Hello World'
　　 		}]
　　 	}],
　　 	onOk: function(){
            if (this._.selectedElement) {

            }

            var mytxt = CKEDITOR.tools.trim(this.getValueOf("cb", "mytxt"));
　　 		editor.insertHtml(mytxt);
　　 	}
	};
});

