#!/usr/bin/env python3
import argparse
import os
import sys

try:
    import uno
    from com.sun.star.beans import PropertyValue
except Exception as exc:
    sys.stderr.write("无法导入 UNO 模块: %s\n" % exc)
    sys.exit(2)

WRITER_EXTS = {".doc", ".docx", ".odt", ".rtf", ".txt", ".html", ".htm"}
CALC_EXTS = {".xls", ".xlsx", ".ods", ".csv", ".tsv"}
IMPRESS_EXTS = {".ppt", ".pptx", ".odp"}

FILTERS = {
    "pdf": {
        "writer": "writer_pdf_Export",
        "calc": "calc_pdf_Export",
        "impress": "impress_pdf_Export",
    },
    "docx": {
        "writer": "MS Word 2007 XML",
    },
    "xlsx": {
        "calc": "Calc MS Excel 2007 XML",
    },
    "pptx": {
        "impress": "Impress MS PowerPoint 2007 XML",
    },
}


def detect_doc_type(input_path):
    ext = os.path.splitext(input_path)[1].lower()
    if ext in WRITER_EXTS:
        return "writer"
    if ext in CALC_EXTS:
        return "calc"
    if ext in IMPRESS_EXTS:
        return "impress"
    return ""


def build_filter_name(output_format, doc_type):
    output_format = output_format.lower()
    if output_format not in FILTERS:
        return ""
    return FILTERS[output_format].get(doc_type, "")


def convert(host, port, input_path, output_path, output_format):
    doc_type = detect_doc_type(input_path)
    if not doc_type:
        raise RuntimeError("无法识别输入文件类型")

    filter_name = build_filter_name(output_format, doc_type)
    if not filter_name:
        raise RuntimeError("输出格式与输入类型不匹配")

    local_ctx = uno.getComponentContext()
    resolver = local_ctx.ServiceManager.createInstanceWithContext(
        "com.sun.star.bridge.UnoUrlResolver", local_ctx
    )
    ctx = resolver.resolve(
        "uno:socket,host=%s,port=%s;urp;StarOffice.ComponentContext" % (host, port)
    )
    smgr = ctx.ServiceManager
    desktop = smgr.createInstanceWithContext("com.sun.star.frame.Desktop", ctx)

    input_url = uno.systemPathToFileUrl(os.path.abspath(input_path))
    output_url = uno.systemPathToFileUrl(os.path.abspath(output_path))

    load_props = (PropertyValue("Hidden", 0, True, 0),)
    document = desktop.loadComponentFromURL(input_url, "_blank", 0, load_props)
    if document is None:
        raise RuntimeError("无法打开输入文件")

    try:
        store_props = (
            PropertyValue("FilterName", 0, filter_name, 0),
            PropertyValue("Overwrite", 0, True, 0),
        )
        document.storeToURL(output_url, store_props)
    finally:
        document.close(True)


def main():
    parser = argparse.ArgumentParser(description="UNO 文档转换脚本")
    parser.add_argument("--host", required=True, help="UNO 监听地址")
    parser.add_argument("--port", required=True, help="UNO 监听端口")
    parser.add_argument("--input", required=True, help="输入文件路径")
    parser.add_argument("--output", required=True, help="输出文件路径")
    parser.add_argument("--format", required=True, help="输出格式")
    args = parser.parse_args()

    try:
        convert(args.host, args.port, args.input, args.output, args.format)
    except Exception as exc:
        sys.stderr.write("UNO 转换失败: %s\n" % exc)
        sys.exit(1)


if __name__ == "__main__":
    main()
