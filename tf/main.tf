module "stackgen_c50b5e78-f222-4e49-b636-4f217a9acdd8" {
  source                       = "./modules/aws_s3"
  block_public_access          = true
  bucket_name                  = "whatver"
  bucket_policy                = ""
  enable_versioning            = true
  enable_website_configuration = false
  sse_algorithm                = "aws:kms"
  tags                         = {}
  website_error_document       = "404.html"
  website_index_document       = "index.html"
}

