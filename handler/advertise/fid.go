package advertise

import(
    "github.com/fecshopsoft/fec-go/util"
    "github.com/gin-gonic/gin"
    "net/http"
    "github.com/fecshopsoft/fec-go/helper"
    "github.com/fecshopsoft/fec-go/shell/model"
    "github.com/fecshopsoft/fec-go/initialization"
    "context"
    "strconv"
    "log"
    // "reflect"
    "encoding/json"
    "github.com/olivere/elastic"
    "github.com/fecshopsoft/fec-go/db/esdb"
)

/**
 * 1.对于common，需要价格websiteId的权限，给下面的用户，通过设置权限，来限制type==3的用户查看的websiteId
 * 2.在router部分加一个middleware，如果type==3，通过用户的parent_id，然后查看相应
 *   的common admin的所有权限的 active website_id，然后去权限表里面查看允许的website_id，然后取他们
 *   的交集，作为当前用户的可用websiteId
 * 2.1.如果type=2，则找到可用的websiteId
 * 2.2.如果type=1，如果request传递了common admin id，则使用该，否则随机找一个common admin作为 own_id
 * 3.对于type=1，服务端需要返回common own id给vue
 * 4.vue将 common own id保存到storage，并设置相应的关联的变量的值，
 * 5.vue的每次请求，都要从storage中取值，如果存在，则在放到请求里面，这个主要是针对type=1的超级admin用户
 * 6.对于vue的请求，如果是超级admin，那么就会使用传递的own_admin，如果没有，则随机找一个admin
 * 7.vue端需要有全部common own id 列表，可以通过select切换，切换后，需要重新 将 common own id保存到storage
 * 8.如果websiteId没有选择，则将可用的websiteId里面找一个可用的，然后将可用的
 *   websiteIds 和 选择的websiteId 发送给vue。vue进行刷新
 */

func FidList(c *gin.Context){
    
    defaultPageNum:= c.GetString("defaultPageNum")
    defaultPageCount := c.GetString("defaultPageCount")
    page, _  := strconv.Atoi(c.DefaultQuery("page", defaultPageNum))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultPageCount))
    sort     := c.DefaultQuery("sort", "")
    sort_dir := c.DefaultQuery("sort_dir", "")
    
    service_date_str_begin := c.DefaultQuery("service_date_str_begin", "")
    service_date_str_end := c.DefaultQuery("service_date_str_end", "")
    fid := c.DefaultQuery("fid", "")
    fec_content := c.DefaultQuery("fec_content", "")
    fec_design := c.DefaultQuery("fec_design", "")
    uv_begin := c.DefaultQuery("uv_begin", "")
    uv_end := c.DefaultQuery("uv_end", "")
    // 搜索条件
    q := elastic.NewBoolQuery()
    // service_date_str 范围搜索
    if service_date_str_begin != "" || service_date_str_end != "" {
        newRangeQuery := elastic.NewRangeQuery("service_date_str")
        if service_date_str_begin != "" {
            newRangeQuery.Gte(service_date_str_begin)
        }
        if service_date_str_end != "" {
            newRangeQuery.Lt(service_date_str_end)
        }
        q = q.Must(newRangeQuery)
    }
    // fid 搜索
    if fid != "" {
        q = q.Must(elastic.NewTermQuery("fid", fid))
    }
    // fec_content 搜索
    if fec_content != "" {
        q = q.Must(elastic.NewTermQuery("fec_content", fec_content))
    }
    // fec_design 搜索
    if fec_design != "" {
        q = q.Must(elastic.NewTermQuery("fec_design", fec_design))
    }
    // uv 范围搜索
    if uv_begin != "" || uv_end != "" {
        newRangeQuery := elastic.NewRangeQuery("uv")
        if uv_begin != "" {
            uvBeginInt, _  := strconv.Atoi(uv_begin)
            newRangeQuery.Gte(uvBeginInt)
        }
        if uv_end != "" {
            uvEndInt, _  := strconv.Atoi(uv_end)
            newRangeQuery.Lt(uvEndInt)
        }
        q = q.Must(newRangeQuery)
    }
    //////   chosen_own_id,  selectOwnIds, 
   chosen_website_id, selectWebsiteIds, err := ActiveWebsite(c)
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    if chosen_website_id == "" {
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult("website id is empty"))
        return
    }
    // 查询出来当前的员工和设计者
    contentNames, designNames, err := getContentAndDesign(c)
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    // 添加website_id 搜索条件
    q = q.Must(elastic.NewTermQuery("website_id", chosen_website_id))
    
    // esIndexName := helper.GetEsIndexName(chosen_website_id)
    esAdvertiseFidTypeName :=  helper.GetEsAdvertiseFidTypeName()
    esIndexName := helper.GetEsIndexNameByType(esAdvertiseFidTypeName)
    client, err := esdb.Client()
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    
    // q = q.Must(elastic.NewTermQuery("country_code", "Safari"))
    // q = q.Must(elastic.NewRangeQuery("pv").From(3).To(60))
    // q = q.Must(elastic.NewRangeQuery("service_date_str").Gt("2018-04-20").Lt("2018-04-21"))
    //termQuery := elastic.NewTermQuery("country_code", "Safari")
    // termQuery := elastic.NewRangeQuery("country_code", "Safari")
    //rangeQuery := NewRangeQuery("pv").Gt(3)
    log.Println(8888888888888)
    log.Println(esIndexName)
    log.Println(esAdvertiseFidTypeName)
    log.Println(page-1)
    log.Println(limit)
    log.Println(sort)
    search := client.Search().
        Index(esIndexName).        // search in index "twitter"
        Type(esAdvertiseFidTypeName).
        Query(q).
        From((page-1)*limit).Size(limit).
        Pretty(true)
    if sort != "" {
        if sort_dir == "ascending" {
            search.Sort(sort, true)
        } else {
            search.Sort(sort, false)
        }
    }
    searchResult, err := search.Do(context.Background())   
    /*
    searchResult, err := client.Search().
        Index(esIndexName).        // search in index "twitter"
        Type(esAdvertiseFidTypeName).
        Query(q).        // specify the query
        //Sort("user", true).      // sort by "user" field, ascending
        From(0).Size(10).        // take documents 0-9
        Pretty(true).            // pretty print request and response JSON
        Do(context.Background()) // execute
    */
    if err != nil{
        log.Println(err.Error())
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    
    var designGroupArr  []helper.VueSelectOps
    var contentGroupArr []helper.VueSelectOps
    var marketGroupArr  []helper.VueSelectOps
    
    var ts []model.AdvertiseFidValue
    if searchResult.Hits.TotalHits > 0 {
        // Iterate through results
        for _, hit := range searchResult.Hits.Hits {
            // hit.Index contains the name of the index

            // Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
            var advertiseFid model.AdvertiseFidValue
            err := json.Unmarshal(*hit.Source, &advertiseFid)
            if err != nil{
                c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
                return
            }
            ts = append(ts, advertiseFid)
            
            fecDesignInt64, _ := helper.Int64(advertiseFid.FecDesign)
            designGroupVal := initialization.CustomerIdWithUsername[fecDesignInt64]
            designGroupArr = append(designGroupArr, helper.VueSelectOps{Key: fecDesignInt64, DisplayName: designGroupVal})
            
            fecContent64, _ := helper.Int64(advertiseFid.FecContent)
            contentGroupVal := initialization.CustomerIdWithUsername[fecContent64]
            contentGroupArr = append(contentGroupArr, helper.VueSelectOps{Key: fecContent64, DisplayName: contentGroupVal})
            
            fecMarketGroup64, _ := helper.Int64(advertiseFid.FecMarketGroup)
            marketGroupVal := initialization.MarketGroupIdWithName[fecMarketGroup64]
            marketGroupArr = append(marketGroupArr, helper.VueSelectOps{Key: fecMarketGroup64, DisplayName: marketGroupVal})
            
        }
    }
    
    siteNameOptions, siteImgUrls, err := getSiteNameAndImgUrls(c, selectWebsiteIds)
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    // 生成返回结果
    result := util.BuildSuccessResult(gin.H{
        "success": "success",
        "total": searchResult.Hits.TotalHits,
        "items": ts,
        "chosen_website_id": chosen_website_id,
        "selectWebsiteIds": selectWebsiteIds,
        "siteIdOptions": siteNameOptions,
        "siteImgUrls": siteImgUrls,
        "designGroupOps": designGroupArr,
        "contentGroupOps": contentGroupArr,
        "marketGroupOps": marketGroupArr,
        "contentSelectOps": contentNames,
        "designSelectOps": designNames,
    })
    
    // 返回json
    c.JSON(http.StatusOK, result)
}

// 得到 trend  info
func FidTrendInfo(c *gin.Context){
    fid := c.DefaultQuery("fid", "")
    if fid == ""{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult("request get param fid can not empty"))
        return
    }
    service_date_str := c.DefaultQuery("service_date_str", "")
    if service_date_str == ""{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult("request get param service_date_str can not empty"))
        return
    }
    endtimestamps := helper.GetTimestampsByDate(service_date_str)
    //timestamps := helper.DateTimestamps()
    begintimestamps := endtimestamps - 86400 * helper.TrendDays
    preMonthDateStr := helper.GetDateTimeUtcByTimestamps(begintimestamps)
    
    // 搜索条件
    q := elastic.NewBoolQuery()
    // 加入时间
    newRangeQuery := elastic.NewRangeQuery("service_date_str")
    newRangeQuery.Gte(preMonthDateStr)
    newRangeQuery.Lte(service_date_str)
    log.Println(preMonthDateStr)
    log.Println(service_date_str)
    q = q.Must(newRangeQuery)
    // 加入浏览器
    q = q.Must(elastic.NewTermQuery("fid", fid))
    website_id, err := GetReqWebsiteId(c)
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    q = q.Must(elastic.NewTermQuery("website_id", website_id))
    // esIndexName := helper.GetEsIndexName(website_id)
    esAdvertiseFidTypeName :=  helper.GetEsAdvertiseFidTypeName()
    esIndexName := helper.GetEsIndexNameByType(esAdvertiseFidTypeName)
    client, err := esdb.Client()
    if err != nil{
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    search := client.Search().
        Index(esIndexName).        // search in index "twitter"
        Type(esAdvertiseFidTypeName).
        Query(q).
        From(0).Size(9999).
        Pretty(true)
    searchResult, err := search.Do(context.Background())   
    
    if err != nil{
        log.Println(err.Error())
        c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
        return
    }
    
    pvTrend := make(map[string]int64)
    uvTrend := make(map[string]int64)
    ipCountTrend := make(map[string]int64)
    staySecondsTrend := make(map[string]float64)
    staySecondsRateTrend := make(map[string]float64)
    PvRateTrend := make(map[string]float64)
    JumpOutCountTrend := make(map[string]int64)
    DropOutCountTrend := make(map[string]int64)
    JumpOutRateTrend := make(map[string]float64)
    DropOutRateTrend := make(map[string]float64)
    CartCountTrend := make(map[string]int64)
    OrderCountTrend := make(map[string]int64)
    SuccessOrderCountTrend := make(map[string]int64)
    SuccessOrderNoCountTrend := make(map[string]int64)
    OrderNoCountTrend := make(map[string]int64)
    OrderPaymentRateTrend := make(map[string]float64)
    
    OrderAmountTrend := make(map[string]float64)
    SuccessOrderAmountTrend := make(map[string]float64)

    IsReturnTrend := make(map[string]int64)
    IsReturnRateTrend := make(map[string]float64)
    SkuSaleRateTrend := make(map[string]float64)
    
    
    RegisterCountTrend := make(map[string]int64)
    LoginCountTrend := make(map[string]int64)
    CategoryCountTrend := make(map[string]int64)
    SkuCountTrend := make(map[string]int64)
    SearchCountTrend := make(map[string]int64)
    success_order_c_success_uv_rateTrend := make(map[string]float64)
    success_order_c_all_uv_rateTrend := make(map[string]float64)
    if searchResult.Hits.TotalHits > 0 {
        // Iterate through results
        for _, hit := range searchResult.Hits.Hits {
            // hit.Index contains the name of the index

            // Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
            var advertiseFid model.AdvertiseFidValue
            err := json.Unmarshal(*hit.Source, &advertiseFid)
            if err != nil{
                c.AbortWithStatusJSON(http.StatusOK, util.BuildFailResult(err.Error()))
                return
            }
            serviceDateStr := advertiseFid.ServiceDateStr
            
            RegisterCountTrend[serviceDateStr] = advertiseFid.RegisterCount
            LoginCountTrend[serviceDateStr] = advertiseFid.LoginCount
            CategoryCountTrend[serviceDateStr] = advertiseFid.CategoryCount
            SkuCountTrend[serviceDateStr] = advertiseFid.SkuCount
            SearchCountTrend[serviceDateStr] = advertiseFid.SearchCount
            success_order_c_success_uv_rateTrend[serviceDateStr] = advertiseFid.SuccessOrderCSuccessUvRate
            success_order_c_all_uv_rateTrend[serviceDateStr] = advertiseFid.SuccessOrderCAllUvRate
            // pvTrend
            pvTrend[serviceDateStr] = advertiseFid.Pv
            // uvTrend
            uvTrend[serviceDateStr] = advertiseFid.Uv
            ipCountTrend[serviceDateStr] = advertiseFid.IpCount
            // staySecondsTrend
            staySecondsTrend[serviceDateStr] = advertiseFid.StaySeconds
            // staySecondsRateTrend
            staySecondsRateTrend[serviceDateStr] = advertiseFid.StaySecondsRate
            PvRateTrend[serviceDateStr] = advertiseFid.PvRate
            JumpOutCountTrend[serviceDateStr] = advertiseFid.JumpOutCount
            DropOutCountTrend[serviceDateStr] = advertiseFid.DropOutCount
            JumpOutRateTrend[serviceDateStr] = advertiseFid.JumpOutRate
            DropOutRateTrend[serviceDateStr] = advertiseFid.DropOutRate
            CartCountTrend[serviceDateStr] = advertiseFid.CartCount
            OrderCountTrend[serviceDateStr] = advertiseFid.OrderCount
            SuccessOrderCountTrend[serviceDateStr] = advertiseFid.SuccessOrderCount
            SuccessOrderNoCountTrend[serviceDateStr] = advertiseFid.SuccessOrderNoCount
            OrderNoCountTrend[serviceDateStr] = advertiseFid.OrderNoCount
            OrderPaymentRateTrend[serviceDateStr] = advertiseFid.OrderPaymentRate
            OrderAmountTrend[serviceDateStr] = advertiseFid.OrderAmount
            SuccessOrderAmountTrend[serviceDateStr] = advertiseFid.SuccessOrderAmount
            
            IsReturnTrend[serviceDateStr] = advertiseFid.IsReturn
            IsReturnRateTrend[serviceDateStr] = advertiseFid.IsReturnRate
            SkuSaleRateTrend[serviceDateStr] = advertiseFid.SkuSaleRate
        }
    }
    // 生成返回结果
    result := util.BuildSuccessResult(gin.H{
        "success": "success",
        "trend": gin.H{
            "pv": pvTrend,
            "uv": uvTrend,
            "ip_count": ipCountTrend,
            "stay_seconds": staySecondsTrend,
            "stay_seconds_rate": staySecondsRateTrend,
            "pv_rate": PvRateTrend,
            "jump_out_count": JumpOutCountTrend,
            "drop_out_count": DropOutCountTrend,
            "jump_out_rate": JumpOutRateTrend,
            "drop_out_rate": DropOutRateTrend,
            "cart_count": CartCountTrend,
            "order_count": OrderCountTrend,
            "success_order_count": SuccessOrderCountTrend,
            "success_order_no_count": SuccessOrderNoCountTrend,
            "order_no_count": OrderNoCountTrend,
            "order_payment_rate": OrderPaymentRateTrend,
            "order_amount": OrderAmountTrend,
            "success_order_amount": SuccessOrderAmountTrend,
            "is_return": IsReturnTrend,
            "is_return_rate": IsReturnRateTrend,
            "sku_sale_rate": SkuSaleRateTrend,
            "register_count": RegisterCountTrend,
            "login_count": LoginCountTrend,
            "category_count": CategoryCountTrend,
            "sku_count": SkuCountTrend,
            "search_count": SearchCountTrend,
            "success_order_c_success_uv_rate": success_order_c_success_uv_rateTrend,
            "success_order_c_all_uv_rate": success_order_c_all_uv_rateTrend,
        },
    })
    // 返回json
    c.JSON(http.StatusOK, result)
    
}










